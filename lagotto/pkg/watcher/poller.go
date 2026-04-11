package watcher

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	truffleaws "github.com/scttfrdmn/spore-host/truffle/pkg/aws"
)

// Poller checks instance capacity for active watches.
type Poller struct {
	truffle  *truffleaws.Client
	store    *Store
	notifier *Notifier // nil = skip notifications
	spawner  *Spawner  // nil = skip auto-spawn
	holder   *Holder   // nil = skip capacity reservations
	verbose  bool
}

// PollerOpts configures optional Poller dependencies.
type PollerOpts struct {
	Notifier *Notifier
	Spawner  *Spawner
	Holder   *Holder
}

// NewPoller creates a Poller backed by a truffle client and DynamoDB store.
func NewPoller(truffle *truffleaws.Client, store *Store, verbose bool, opts ...PollerOpts) *Poller {
	p := &Poller{
		truffle: truffle,
		store:   store,
		verbose: verbose,
	}
	if len(opts) > 0 {
		p.notifier = opts[0].Notifier
		p.spawner = opts[0].Spawner
		p.holder = opts[0].Holder
	}
	return p
}

// PollAll loads all active watches and polls each one.
// Returns all matches found across all watches.
func (p *Poller) PollAll(ctx context.Context) ([]MatchResult, error) {
	watches, err := p.store.ListActiveWatches(ctx)
	if err != nil {
		return nil, fmt.Errorf("load active watches: %w", err)
	}
	if p.verbose {
		fmt.Fprintf(os.Stderr, "Polling %d active watches\n", len(watches))
	}

	// Group watches by region set to deduplicate API calls
	type regionKey struct {
		regions string // sorted, joined
		pattern string
		spot    bool
	}
	type regionGroup struct {
		regions []string
		pattern string
		spot    bool
		watches []*Watch
	}

	groups := make(map[regionKey]*regionGroup)
	for i := range watches {
		w := &watches[i]
		key := regionKey{
			regions: strings.Join(w.Regions, ","),
			pattern: w.InstanceTypePattern,
			spot:    w.Spot,
		}
		if g, ok := groups[key]; ok {
			g.watches = append(g.watches, w)
		} else {
			groups[key] = &regionGroup{
				regions: w.Regions,
				pattern: w.InstanceTypePattern,
				spot:    w.Spot,
				watches: []*Watch{w},
			}
		}
	}

	var allMatches []MatchResult

	for _, g := range groups {
		matches, err := p.pollGroup(ctx, g.regions, g.pattern, g.spot, g.watches)
		if err != nil {
			// Log but don't fail the entire poll cycle
			fmt.Fprintf(os.Stderr, "Warning: poll failed for pattern %q: %v\n", g.pattern, err)
			continue
		}
		allMatches = append(allMatches, matches...)
	}

	return allMatches, nil
}

// PollWatch runs a single poll cycle for one watch. Useful for testing.
func (p *Poller) PollWatch(ctx context.Context, w *Watch) ([]MatchResult, error) {
	return p.pollGroup(ctx, w.Regions, w.InstanceTypePattern, w.Spot, []*Watch{w})
}

func (p *Poller) pollGroup(ctx context.Context, regions []string, pattern string, spot bool, watches []*Watch) ([]MatchResult, error) {
	// Convert pattern to regex (support wildcards like "p5.*")
	regexPattern := wildcardToRegex(pattern)
	matcher, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("compile pattern %q: %w", pattern, err)
	}

	if p.verbose {
		fmt.Fprintf(os.Stderr, "Searching for %q across %d regions (spot=%v)\n", pattern, len(regions), spot)
	}

	// Search for instance types
	results, err := p.truffle.SearchInstanceTypes(ctx, regions, matcher, truffleaws.FilterOptions{
		IncludeAZs: true,
		Verbose:    p.verbose,
	})
	if err != nil {
		return nil, fmt.Errorf("search instance types: %w", err)
	}

	if len(results) == 0 {
		// No capacity found; update last polled timestamps
		for _, w := range watches {
			_ = p.store.UpdateLastPolled(ctx, w.WatchID)
		}
		return nil, nil
	}

	// Get Spot pricing if needed
	var spotResults []truffleaws.SpotPriceResult
	if spot {
		spotResults, err = p.truffle.GetSpotPricing(ctx, results, truffleaws.SpotOptions{
			OnlyActive: true,
			Verbose:    p.verbose,
		})
		if err != nil {
			return nil, fmt.Errorf("get spot pricing: %w", err)
		}
	}

	// Evaluate each watch against the results
	now := time.Now().UTC()
	var allMatches []MatchResult

	for _, w := range watches {
		var bestMatch *MatchResult

		if spot && len(spotResults) > 0 {
			// Evaluate Spot results
			for i := range spotResults {
				candidate := MatchCandidate{SpotPrice: &spotResults[i]}
				// Find the corresponding InstanceTypeResult
				for j := range results {
					if results[j].InstanceType == spotResults[i].InstanceType && results[j].Region == spotResults[i].Region {
						candidate.InstanceType = results[j]
						break
					}
				}
				if m := Evaluate(w, candidate); m != nil {
					m.MatchedAt = now
					if bestMatch == nil || m.Price < bestMatch.Price {
						bestMatch = m
					}
				}
			}
		} else if !spot {
			// Evaluate On-Demand results
			for i := range results {
				candidate := MatchCandidate{InstanceType: results[i]}
				if m := Evaluate(w, candidate); m != nil {
					m.MatchedAt = now
					if bestMatch == nil || m.Price < bestMatch.Price {
						bestMatch = m
					}
				}
			}
		}

		if bestMatch != nil {
			if p.verbose {
				fmt.Fprintf(os.Stderr, "Match found for watch %s: %s in %s at $%.4f/hr\n",
					w.WatchID, bestMatch.InstanceType, bestMatch.Region, bestMatch.Price)
			}

			// Execute action (before notification so we can include result details)
			switch w.Action {
			case ActionSpawn:
				if p.spawner != nil {
					if err := p.spawner.Spawn(ctx, w, bestMatch); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: auto-spawn failed for %s: %v\n", w.WatchID, err)
						bestMatch.ActionTaken = "spawn_failed"
					}
				} else {
					bestMatch.ActionTaken = "notified"
				}
			case ActionHold:
				if p.holder != nil {
					if err := p.holder.Hold(ctx, w, bestMatch); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: hold failed for %s: %v\n", w.WatchID, err)
						bestMatch.ActionTaken = "hold_failed"
					}
				} else {
					bestMatch.ActionTaken = "notified"
				}
			default:
				bestMatch.ActionTaken = "notified"
			}

			// Record the match
			if err := p.store.RecordMatch(ctx, w, bestMatch); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to record match for %s: %v\n", w.WatchID, err)
			}

			// Send notifications
			if p.notifier != nil {
				if err := p.notifier.Notify(ctx, w, bestMatch); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: notification failed for %s: %v\n", w.WatchID, err)
				}
			}

			// Update watch status to matched
			if err := p.store.UpdateWatchStatus(ctx, w.WatchID, StatusMatched); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to update watch status for %s: %v\n", w.WatchID, err)
			}
			allMatches = append(allMatches, *bestMatch)
		} else {
			_ = p.store.UpdateLastPolled(ctx, w.WatchID)
		}
	}

	return allMatches, nil
}

// wildcardToRegex converts shell-style wildcards to regex.
// e.g., "p5.*" -> "^p5\..*$", "g5.xlarge" -> "^g5\.xlarge$"
func wildcardToRegex(pattern string) string {
	// If it already looks like a regex, use as-is
	if strings.ContainsAny(pattern, "^$()[]{}+\\") {
		return pattern
	}
	// Escape dots, convert * to .*
	escaped := strings.ReplaceAll(pattern, ".", "\\.")
	escaped = strings.ReplaceAll(escaped, "*", ".*")
	return "^" + escaped + "$"
}
