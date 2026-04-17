package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	schedulertypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/google/uuid"
	"github.com/scttfrdmn/spore-host/lagotto/pkg/watcher"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	watchRegions     []string
	watchSpot        bool
	watchMaxPrice    float64
	watchAction      string
	watchTTL         string
	watchNotify      []string
	watchSpawnConfig string
)

var watchCmd = &cobra.Command{
	Use:   "watch <instance-type-pattern>",
	Short: "Create a capacity watch for an instance type",
	Long: `Watch for EC2 instance availability across regions and AZs.

The pattern supports wildcards: "p5.*" matches all p5 sizes, "g5.xlarge" is exact.
When capacity is found matching your criteria, the configured action is taken.`,
	Args: cobra.ExactArgs(1),
	RunE: runWatch,
}

func init() {
	rootCmd.AddCommand(watchCmd)

	watchCmd.Flags().StringSliceVarP(&watchRegions, "regions", "r", nil, "Regions to watch (comma-separated; empty = all enabled)")
	watchCmd.Flags().BoolVar(&watchSpot, "spot", false, "Watch for Spot capacity (default: On-Demand)")
	watchCmd.Flags().Float64Var(&watchMaxPrice, "max-price", 0, "Maximum acceptable price per hour (0 = any)")
	watchCmd.Flags().StringVar(&watchAction, "action", "notify", "Action on match: notify, spawn, hold")
	watchCmd.Flags().StringVar(&watchTTL, "ttl", "24h", "How long to keep watching (e.g., 24h, 7d)")
	watchCmd.Flags().StringSliceVar(&watchNotify, "notify", nil, "Notification channels (e.g., email:user@example.com, webhook:https://...)")
	watchCmd.Flags().StringVar(&watchSpawnConfig, "spawn-config", "", "YAML file with spawn LaunchConfig (required for --action spawn)")
}

func runWatch(cmd *cobra.Command, args []string) error {
	pattern := args[0]
	ctx := context.Background()

	// Parse TTL
	ttl, err := time.ParseDuration(watchTTL)
	if err != nil {
		ttl, err = parseDuration(watchTTL)
		if err != nil {
			return fmt.Errorf("invalid TTL %q: %w", watchTTL, err)
		}
	}

	// Validate action
	action := watcher.ActionMode(watchAction)
	switch action {
	case watcher.ActionNotify, watcher.ActionSpawn, watcher.ActionHold:
	default:
		return fmt.Errorf("invalid action %q: must be notify, spawn, or hold", watchAction)
	}

	// Load spawn config if action is spawn
	var launchConfigJSON []byte
	if action == watcher.ActionSpawn {
		if watchSpawnConfig == "" {
			return fmt.Errorf("--spawn-config is required when --action=spawn")
		}
		launchConfigJSON, err = loadSpawnConfig(watchSpawnConfig)
		if err != nil {
			return fmt.Errorf("load spawn config: %w", err)
		}
	}

	// Parse notify channels
	channels, err := parseNotifyChannels(watchNotify)
	if err != nil {
		return err
	}

	// Get caller identity for user_id
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("get caller identity: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	w := &watcher.Watch{
		WatchID:             "w-" + uuid.New().String()[:8],
		UserID:              *identity.Arn,
		Status:              watcher.StatusActive,
		InstanceTypePattern: pattern,
		Regions:             watchRegions,
		Spot:                watchSpot,
		MaxPrice:            watchMaxPrice,
		Action:              action,
		NotifyChannels:      channels,
		LaunchConfigJSON:    launchConfigJSON,
		CreatedAt:           now,
		UpdatedAt:           now,
		ExpiresAt:           expiresAt,
		TTLTimestamp:        expiresAt.Unix(),
	}

	store := watcher.NewStore(cfg, watchesTable, historyTable)
	if err := store.PutWatch(ctx, w); err != nil {
		return fmt.Errorf("create watch: %w", err)
	}

	// Try to enable the polling schedule (best-effort; fails gracefully if not deployed)
	if err := enablePollingSchedule(ctx, cfg); err != nil && verbose {
		fmt.Fprintf(os.Stderr, "Note: could not enable polling schedule: %v\n", err)
	}

	if getOutputFormat() == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(w)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created watch %s\n", w.WatchID)
	fmt.Fprintf(cmd.OutOrStdout(), "  Pattern:  %s\n", w.InstanceTypePattern)
	fmt.Fprintf(cmd.OutOrStdout(), "  Regions:  %v\n", displayRegions(w.Regions))
	fmt.Fprintf(cmd.OutOrStdout(), "  Spot:     %v\n", w.Spot)
	if w.MaxPrice > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Max price: $%.4f/hr\n", w.MaxPrice)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  Action:   %s\n", w.Action)
	if action == watcher.ActionSpawn {
		fmt.Fprintf(cmd.OutOrStdout(), "  Spawn config: %s\n", watchSpawnConfig)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  Expires:  %s\n", w.ExpiresAt.Format(time.RFC3339))
	return nil
}

// loadSpawnConfig reads a YAML file and converts it to JSON bytes for storage.
func loadSpawnConfig(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	// Parse YAML to validate structure
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(data, &configMap); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	// Re-marshal as JSON for DynamoDB storage
	jsonBytes, err := json.Marshal(configMap)
	if err != nil {
		return nil, fmt.Errorf("marshal to JSON: %w", err)
	}
	return jsonBytes, nil
}

// enablePollingSchedule re-enables the EventBridge schedule if it was self-disabled.
func enablePollingSchedule(ctx context.Context, cfg aws.Config) error {
	scheduleName := os.Getenv("LAGOTTO_SCHEDULE_NAME")
	if scheduleName == "" {
		scheduleName = "lagotto-capacity-poller"
	}

	client := scheduler.NewFromConfig(cfg)

	current, err := client.GetSchedule(ctx, &scheduler.GetScheduleInput{
		Name: aws.String(scheduleName),
	})
	if err != nil {
		// Schedule doesn't exist (not deployed yet) — that's fine
		return nil
	}

	if current.State == schedulertypes.ScheduleStateDisabled {
		_, err = client.UpdateSchedule(ctx, &scheduler.UpdateScheduleInput{
			Name:               current.Name,
			ScheduleExpression: current.ScheduleExpression,
			FlexibleTimeWindow: current.FlexibleTimeWindow,
			Target:             current.Target,
			State:              schedulertypes.ScheduleStateEnabled,
		})
		if err != nil {
			return fmt.Errorf("enable schedule: %w", err)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "Enabled polling schedule %s\n", scheduleName)
		}
	}
	return nil
}

func displayRegions(regions []string) string {
	if len(regions) == 0 {
		return "(all enabled)"
	}
	return fmt.Sprintf("%v", regions)
}

func parseNotifyChannels(raw []string) ([]watcher.NotifyChannel, error) {
	var channels []watcher.NotifyChannel
	for _, s := range raw {
		parts := splitFirst(s, ':')
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid notify format %q: expected type:target (e.g., email:user@example.com)", s)
		}
		ch := watcher.NotifyChannel{Type: parts[0], Target: parts[1]}
		switch ch.Type {
		case "email", "sns":
		case "webhook":
			if err := watcher.ValidateWebhookURL(ch.Target); err != nil {
				return nil, fmt.Errorf("invalid webhook URL: %w", err)
			}
		default:
			return nil, fmt.Errorf("invalid notify type %q: must be email, webhook, or sns", ch.Type)
		}
		channels = append(channels, ch)
	}
	return channels, nil
}

func splitFirst(s string, sep byte) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}
	unit := s[len(s)-1]
	val := s[:len(s)-1]
	var n int
	if _, err := fmt.Sscanf(val, "%d", &n); err != nil {
		return 0, fmt.Errorf("invalid duration number: %s", val)
	}
	switch unit {
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'm':
		return time.Duration(n) * time.Minute, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %c", unit)
	}
}
