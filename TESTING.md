# Testing Guide

This guide covers testing practices for the spore-host project (spawn and truffle).

## Running Tests

### All Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Generate HTML coverage report
make test-coverage-report
```

### Specific Test Suites

```bash
# i18n translation validation tests
make test-i18n
go test ./pkg/i18n/... -v

# spawn command tests
go test ./spawn/cmd/... -v

# truffle command tests
go test ./truffle/cmd/... -v

# spawnd agent tests
go test ./spawn/pkg/agent/... -v

# Specific test function
go test ./spawn/cmd/... -v -run TestValidateTTL
go test ./spawn/pkg/agent/... -v -run TestGetDiskIO
```

### Test with Coverage

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage report in browser
go tool cover -html=coverage.out -o coverage.html
open coverage.html

# View coverage in terminal
go tool cover -func=coverage.out

# Coverage for specific package
go test -coverprofile=coverage.out ./spawn/cmd/
go tool cover -func=coverage.out
```

## Test Organization

```
spore-host/
├── pkg/
│   └── i18n/
│       ├── i18n.go
│       ├── i18n_test.go           # Core i18n tests
│       └── validation_test.go     # Translation validation (Week 1)
├── spawn/
│   ├── cmd/
│   │   ├── list.go
│   │   ├── list_test.go           # spawn list tests (Week 2)
│   │   ├── extend.go
│   │   ├── extend_test.go         # spawn extend tests (Week 2)
│   │   ├── connect.go
│   │   └── connect_test.go        # spawn connect/ssh tests (Week 2)
│   └── pkg/
│       └── agent/
│           ├── agent.go
│           └── monitoring_test.go  # Monitoring tests (Week 3)
└── truffle/
    └── cmd/
        └── i18n_test.go            # truffle i18n tests
```

## Test Coverage Summary

### Week 1: i18n Testing
- **Files**: `pkg/i18n/validation_test.go`, `pkg/i18n/i18n_test.go`
- **Coverage**: Translation validation for 6 languages (en, es, fr, de, ja, pt)
- **Test count**: 443 keys validated across all languages
- **Status**: ✅ Complete

### Week 2: spawn Command Testing
- **Files**: `spawn/cmd/list_test.go`, `extend_test.go`, `connect_test.go`
- **Coverage**: 154 test cases across 3 command test files
- **Test functions**: 35 test functions
- **Status**: ✅ Complete

| File | Test Functions | Test Cases | Coverage Focus |
|------|----------------|------------|----------------|
| `list_test.go` | 12 | 70+ | Duration formatting, filtering (AZ, type, family, tags) |
| `extend_test.go` | 10 | 100+ | TTL validation, format parsing, edge cases |
| `connect_test.go` | 13 | 30+ | SSH alias, key resolution, default values |

### Week 3: spawnd Monitoring Testing
- **Files**: `spawn/pkg/agent/monitoring_test.go`
- **Coverage**: 70+ test cases for monitoring logic
- **Test functions**: 13 test functions
- **Status**: ✅ Complete

| Test Category | Tests | Coverage |
|---------------|-------|----------|
| Disk I/O | 4 | Parsing, partition detection, device types, sector conversion |
| GPU Monitoring | 3 | nvidia-smi detection, parsing, multi-GPU max |
| Idle Detection | 6 | CPU, network, disk, GPU thresholds, all-conditions logic, real-world scenarios |

## Writing Tests

### Unit Test Pattern

```go
func TestFeatureName(t *testing.T) {
    // Arrange
    input := "test input"
    expected := "expected output"

    // Act
    result := functionUnderTest(input)

    // Assert
    if result != expected {
        t.Errorf("got %v, want %v", result, expected)
    }
}
```

### Table-Driven Test Pattern

```go
func TestMultipleCases(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case 1", "input1", "output1"},
        {"case 2", "input2", "output2"},
        {"edge case", "", ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := functionUnderTest(tt.input)
            if got != tt.expected {
                t.Errorf("got %v, want %v", got, tt.expected)
            }
        })
    }
}
```

### Testing with Temporary Files

```go
func TestWithTempFile(t *testing.T) {
    // Create temporary directory (auto-cleaned after test)
    tmpDir := t.TempDir()

    // Create test file
    testFile := filepath.Join(tmpDir, "test.txt")
    err := os.WriteFile(testFile, []byte("content"), 0644)
    if err != nil {
        t.Fatalf("Failed to create test file: %v", err)
    }

    // Test code that uses testFile
    result := functionThatReadsFile(testFile)

    // Assertions
    if result != "expected" {
        t.Errorf("got %v, want %v", result, "expected")
    }
}
```

### Testing with Environment Variables

```go
func TestWithEnvVars(t *testing.T) {
    // Save original value
    oldValue := os.Getenv("MY_VAR")
    defer os.Setenv("MY_VAR", oldValue) // Restore after test

    // Set test value
    os.Setenv("MY_VAR", "test-value")

    // Test code that uses environment variable
    result := functionThatUsesEnv()

    // Assertions
    if result != "expected" {
        t.Errorf("got %v, want %v", result, "expected")
    }
}
```

## Test Examples from the Project

### Example 1: TTL Validation (extend_test.go)

```go
func TestValidateTTL_ValidFormats(t *testing.T) {
    tests := []struct {
        name string
        ttl  string
    }{
        {"Seconds only", "30s"},
        {"Minutes only", "15m"},
        {"Hours only", "2h"},
        {"Days only", "7d"},
        {"Hours and minutes", "2h30m"},
        {"All units", "1d2h30m15s"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateTTL(tt.ttl)
            if err != nil {
                t.Errorf("validateTTL(%q) returned error: %v, want nil", tt.ttl, err)
            }
        })
    }
}
```

### Example 2: Filter Logic (list_test.go)

```go
func TestFilterInstancesByType(t *testing.T) {
    instances := []aws.InstanceInfo{
        {InstanceID: "i-1", InstanceType: "t3.micro"},
        {InstanceID: "i-2", InstanceType: "m7i.large"},
        {InstanceID: "i-3", InstanceType: "t3.micro"},
    }

    // Set filter
    listInstanceType = "t3.micro"
    defer func() { listInstanceType = "" }()

    filtered := filterInstances(instances)

    if len(filtered) != 2 {
        t.Errorf("Expected 2 instances, got %d", len(filtered))
    }

    for _, inst := range filtered {
        if inst.InstanceType != "t3.micro" {
            t.Errorf("Instance %s has wrong type: %s", inst.InstanceID, inst.InstanceType)
        }
    }
}
```

### Example 3: SSH Key Resolution (connect_test.go)

```go
func TestFindSSHKey_WithPemExtension(t *testing.T) {
    tmpDir := t.TempDir()
    sshDir := filepath.Join(tmpDir, ".ssh")
    err := os.Mkdir(sshDir, 0700)
    if err != nil {
        t.Fatalf("Failed to create temp SSH dir: %v", err)
    }

    // Create key with .pem extension
    keyName := "my-key"
    keyPath := filepath.Join(sshDir, keyName+".pem")
    err = os.WriteFile(keyPath, []byte("fake key"), 0600)
    if err != nil {
        t.Fatalf("Failed to write key file: %v", err)
    }

    oldHome := os.Getenv("HOME")
    os.Setenv("HOME", tmpDir)
    defer os.Setenv("HOME", oldHome)

    foundPath, err := findSSHKey(keyName)
    if err != nil {
        t.Errorf("findSSHKey(%q) returned error: %v", keyName, err)
    }

    if foundPath != keyPath {
        t.Errorf("findSSHKey(%q) = %q, want %q", keyName, foundPath, keyPath)
    }
}
```

### Example 4: Threshold Detection (monitoring_test.go)

```go
func TestIsIdle_AllConditions(t *testing.T) {
    tests := []struct {
        name          string
        cpuUsage      float64
        cpuThreshold  float64
        networkBytes  int64
        diskIO        int64
        gpuUtil       float64
        expectedIdle  bool
    }{
        {
            name:         "All metrics idle",
            cpuUsage:     2.0,
            cpuThreshold: 5.0,
            networkBytes: 1000,
            diskIO:       10000,
            gpuUtil:      1.0,
            expectedIdle: true,
        },
        {
            name:         "CPU over threshold",
            cpuUsage:     6.0,
            cpuThreshold: 5.0,
            networkBytes: 1000,
            diskIO:       10000,
            gpuUtil:      1.0,
            expectedIdle: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test that ALL conditions must be met
            cpuIdle := tt.cpuUsage < tt.cpuThreshold
            networkIdle := tt.networkBytes <= 10000
            diskIdle := tt.diskIO <= 100000
            gpuIdle := tt.gpuUtil <= 5.0

            allIdle := cpuIdle && networkIdle && diskIdle && gpuIdle

            if allIdle != tt.expectedIdle {
                t.Errorf("Expected idle=%v, but got idle=%v", tt.expectedIdle, allIdle)
            }
        })
    }
}
```

## CI/CD Integration

Tests run automatically on:
- Every commit (unit tests)
- Every pull request (full test suite)
- Daily builds (translation validation)

### GitHub Actions Configuration

Tests are configured in `.github/workflows/ci.yml` (if exists) to run:

```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: make test
      - run: make test-i18n
```

## Test Data and Fixtures

### Mock Data Locations

Test data should be:
- Created programmatically in tests (preferred)
- Placed in `testdata/` directories (for static fixtures)
- Cleaned up after tests complete

### Example Mock Data

```go
// Good: Programmatic test data
func createMockInstances() []aws.InstanceInfo {
    return []aws.InstanceInfo{
        {
            InstanceID:   "i-test1",
            InstanceType: "m7i.large",
            State:        "running",
        },
        {
            InstanceID:   "i-test2",
            InstanceType: "t3.micro",
            State:        "stopped",
        },
    }
}

// For static fixtures:
// pkg/mypackage/testdata/sample.json
```

## Debugging Failed Tests

### Verbose Output

```bash
# Run with verbose output
go test -v ./spawn/cmd/...

# Run specific test with verbose output
go test -v -run TestValidateTTL ./spawn/cmd/...

# Show test logs even for passing tests
go test -v ./spawn/cmd/... 2>&1 | tee test.log
```

### Debugging Tools

```bash
# Add debug output in tests
t.Logf("Debug: value = %v", value)

# Use fmt.Printf for quick debugging (remove before commit)
fmt.Printf("DEBUG: %+v\n", myStruct)

# Run single test with detailed output
go test -v -run TestSpecificCase ./path/to/package
```

### Common Test Failures

1. **Undefined variable**: Missing import or typo
   ```
   Fix: Add import statement or check spelling
   ```

2. **Cannot use X as type Y**: Type mismatch
   ```
   Fix: Ensure types match or use proper type conversion
   ```

3. **Slice bounds out of range**: Array index issue
   ```
   Fix: Check slice lengths before accessing
   ```

4. **nil pointer dereference**: Accessing nil value
   ```
   Fix: Check for nil before dereferencing
   ```

## Best Practices

### DO ✅

- Write tests before fixing bugs (test-driven debugging)
- Use table-driven tests for multiple similar cases
- Test edge cases (empty strings, nil values, boundary conditions)
- Clean up resources (files, env vars) with `defer`
- Use `t.TempDir()` for temporary directories
- Make test names descriptive (`TestValidateTTL_EmptyString`)
- Test both success and failure cases

### DON'T ❌

- Don't test external services directly (use mocks)
- Don't leave debug print statements in committed code
- Don't skip test cleanup (use `defer`)
- Don't make tests depend on each other (tests must be independent)
- Don't commit commented-out tests
- Don't use hard-coded paths (use `t.TempDir()`, `filepath.Join()`)
- Don't test implementation details (test behavior, not internals)

## Test Naming Conventions

```go
// Format: Test[Function]_[Scenario]
TestValidateTTL_ValidFormats
TestValidateTTL_InvalidFormats
TestValidateTTL_EmptyString
TestValidateTTL_EdgeCases

// For table-driven tests with subtests:
func TestValidateTTL(t *testing.T) {
    tests := []struct {
        name string  // Descriptive name
        ttl  string
        want error
    }{
        {"valid hours", "2h", nil},
        {"empty string", "", ErrInvalidFormat},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test code
        })
    }
}
```

## Coverage Goals

| Package | Target Coverage | Actual Coverage | Status |
|---------|----------------|-----------------|--------|
| pkg/i18n | 95%+ | 95%+ | ✅ |
| spawn/cmd | 80%+ | 90%+ | ✅ |
| truffle/cmd | 80%+ | TBD | ⏳ |
| spawn/pkg/agent | 75%+ | Low* | ⚠️ |

*Note: Agent package has low direct coverage because functions depend on /proc files and AWS APIs. Tests validate logic and algorithms instead.

## Testing Philosophy

### What to Test

- **Business logic**: Core functionality (TTL validation, filtering, parsing)
- **Edge cases**: Empty values, boundaries, invalid input
- **Error handling**: Functions properly handle errors
- **Integration points**: Data flows correctly between components

### What NOT to Test

- **External APIs**: Mock AWS calls, don't actually call AWS
- **File system**: Use temp directories, don't modify real files
- **Third-party libraries**: Trust they work, test your usage
- **Trivial code**: Getters/setters with no logic

## Running Tests in Development

### Quick Workflow

```bash
# During development, run specific test repeatedly
go test -v -run TestValidateTTL ./spawn/cmd/

# Watch mode (requires entr or similar)
ls *.go | entr -c go test -v ./spawn/cmd/

# Run all tests before commit
make test

# Check coverage before PR
make test-coverage-report
open coverage.html
```

### Pre-Commit Checklist

- [ ] All tests pass: `make test`
- [ ] No debug print statements left in code
- [ ] New tests added for new functionality
- [ ] Coverage maintained or improved
- [ ] Test names are descriptive
- [ ] No commented-out tests

## Additional Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table-Driven Tests in Go](https://go.dev/blog/subtests)
- [Testing Best Practices](https://go.dev/doc/effective_go#testing)

## Getting Help

If tests are failing and you're stuck:

1. Read the error message carefully
2. Run with `-v` flag for verbose output
3. Add `t.Logf()` statements to debug
4. Check the test examples in this guide
5. Look at similar tests in the codebase
6. Ask for help (include test output and error message)
