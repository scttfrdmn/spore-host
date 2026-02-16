# CLAUDE.md

## Response Style
- Concise by default. No explanations unless asked.
- No file creation unless explicitly requested.
- Fix bugs silently unless cause is non-obvious.

## Development Workflow

### Planning Mode
**Use plan mode for non-trivial features** - Enter plan mode when:
- Feature spans multiple components or files
- Multiple valid implementation approaches exist
- Architectural decisions need to be made
- User requirements need clarification

**Plan mode process:**
1. Use `EnterPlanMode` tool to explore codebase
2. Launch Explore agents to understand existing patterns
3. Launch Plan agent to design implementation
4. Ask clarifying questions via `AskUserQuestion`
5. Write detailed plan to plan file
6. Use `ExitPlanMode` to get approval before implementing

### Issue Documentation
**Document plans as GitHub issues:**
- Create issue for each major feature/fix before starting
- Include clear scope, acceptance criteria, and tasks
- Link related issues and PRs
- Update issue with implementation notes as you work
- Close with summary comment when complete

**Create milestones for releases:**
- Group related issues under version milestones (e.g., v0.1.0)
- Track progress toward release goals
- Use milestone due dates to guide priorities

**Use labels consistently:**
- `priority:critical`, `priority:high`, `priority:medium`, `priority:low`
- `type:feature`, `type:bug`, `type:refactor`, `type:docs`, `type:test`
- `component:truffle`, `component:spawn`, `component:spawnd`
- Create custom labels as needed for project-specific categories

## Go Standards
- Go 1.21+ with modules
- `gofmt`, `goimports` on all code
- Pass `go vet`, `staticcheck`, `golangci-lint` before done
- Godoc comments on all exported identifiers
- No `panic` except unrecoverable init failures

## Code Style
- Idiomatic short names: `r` for reader, `ctx` for context, `err` for error
- Wrap errors with `fmt.Errorf("operation: %w", err)`
- Return early on errors; avoid deep nesting
- Prefer standard library over dependencies
- Group imports: stdlib, external, internal

## CLI Patterns
- Use `cobra` for CLI structure
- Flags over args when >1 input
- Exit codes: 0=success, 1=error, 2=usage error
- Stderr for errors/logs, stdout for output
- Support `--json` output where applicable

## AWS SDK
- Use `aws-sdk-go-v2`
- Load config with `config.LoadDefaultConfig(ctx)`
- Always pass context for cancellation
- Wrap SDK errors with operation context
- Use pagination helpers for list operations

## Testing
- Minimum 60% coverage, target 80%+
- Table-driven tests as default
- Use `t.Helper()` in test helpers
- Mock AWS with interfaces, not SDK mocks
- Test error paths, not just happy path
- Use `testdata/` for fixtures
- Golden files for complex output verification

## Security
- Never log credentials or tokens
- Use `golang.org/x/crypto` for cryptographic operations
- Validate all external inputs
- Sanitize before logging user-provided data

## Project Structure
- `truffle/` - Instance discovery and quota management
- `spawn/` - EC2 launching and wizard
- Each tool: `cmd/` (commands), `pkg/` (packages)

## Git & GitHub
- Use `gh` CLI for all GitHub operations
- Conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`
- Branch naming: `feat/`, `fix/`, `refactor/` prefixes
- PR per feature/fix; link to issue

## Pre-commit Checks
- Run before every commit: `gofmt`, `go vet`, `staticcheck`
- Smoke tests: `go test -short ./...`
- Use pre-commit hook or `make check`

## Testing Workflow
- `make check` — fast: fmt, vet, lint, short tests
- `make test` — full: all unit tests with coverage
- `make integration` — slow: integration/e2e tests
- `make build` — build binary with version
- Run `make check` before every commit

## Project Tracking
**Single Source of Truth: GitHub**
- Track ALL work via GitHub Issues (not ROADMAP.md or other files)
- Use GitHub Projects for planning/status visualization
- Use GitHub Milestones for release grouping
- Use GitHub Labels for categorization
- Close issues via commit message: `Fixes #123`

**Do NOT maintain:**
- ROADMAP.md or similar planning documents
- TODO lists in markdown files
- External tracking spreadsheets
- Status documents

**Rationale:** Maintaining parallel tracking systems leads to synchronization issues. GitHub provides built-in tools for all project management needs.

## Do Not
- Create README, docs, or configs unless asked
- Maintain ROADMAP.md or other project tracking files (use GitHub Issues/Milestones instead)
- Add dependencies without justification
- Use `interface{}` or `any` without reason
- Ignore returned errors
- Use global state
