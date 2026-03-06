# Makefile for spore-host suite
.PHONY: all build build-all clean install test test-i18n test-coverage test-coverage-report help

# Build for current platform
build:
	@echo "Building spore-host suite..."
	@cd truffle && $(MAKE) build
	@cd spawn && $(MAKE) build
	@echo "✅ Build complete!"
	@echo ""
	@echo "Binaries:"
	@echo "  truffle/bin/truffle"
	@echo "  spawn/bin/spawn"
	@echo "  spawn/bin/spawnd"

# Build for all platforms
build-all:
	@echo "Building spore-host suite for all platforms..."
	@cd truffle && $(MAKE) build-all
	@cd spawn && $(MAKE) build-all
	@echo "✅ Build complete for all platforms!"
	@echo ""
	@echo "See truffle/bin/ and spawn/bin/ for binaries"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@cd truffle && $(MAKE) clean
	@cd spawn && $(MAKE) clean
	@echo "✅ Clean complete"

# Install locally (requires sudo)
install:
	@echo "Installing spore-host suite..."
	@cd truffle && $(MAKE) install
	@cd spawn && $(MAKE) install
	@echo "✅ Installed to /usr/local/bin"

# Run tests
test:
	@echo "Running tests..."
	@cd pkg/i18n && go test .
	@cd truffle && go test ./...
	@cd spawn && go test ./...
	@echo "✅ Tests passed"

# Run i18n validation tests
test-i18n:
	@echo "Testing translations..."
	@echo "Validating translation files..."
	@cd pkg/i18n && go test -v . -run TestTranslation
	@cd pkg/i18n && go test -v . -run TestValidation
	@echo "Testing spawn i18n integration..."
	@cd spawn/cmd && go test -v . -run TestI18n
	@echo "Testing truffle i18n integration..."
	@cd truffle/cmd && go test -v . -run TestI18n
	@echo "✅ i18n tests passed"

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@echo "Testing pkg/i18n..."
	@cd pkg/i18n && go test -coverprofile=../../coverage-pkg.out .
	@echo "Testing spawn..."
	@cd spawn && go test -coverprofile=../coverage-spawn.out ./...
	@echo "Testing truffle..."
	@cd truffle && go test -coverprofile=../coverage-truffle.out ./...
	@echo "✅ Tests passed with coverage"
	@echo ""
	@echo "Coverage summaries:"
	@echo "pkg/i18n:"
	@go tool cover -func=coverage-pkg.out | grep total: || echo "  No coverage data"
	@echo "spawn:"
	@go tool cover -func=coverage-spawn.out | grep total: || echo "  No coverage data"
	@echo "truffle:"
	@go tool cover -func=coverage-truffle.out | grep total: || echo "  No coverage data"

# Generate HTML coverage report
test-coverage-report:
	@echo "Generating coverage reports..."
	@echo "Testing pkg/i18n..."
	@cd pkg/i18n && go test -coverprofile=../../coverage-pkg.out .
	@echo "Testing spawn..."
	@cd spawn && go test -coverprofile=../coverage-spawn.out ./...
	@echo "Testing truffle..."
	@cd truffle && go test -coverprofile=../coverage-truffle.out ./...
	@echo "Generating HTML reports..."
	@go tool cover -html=coverage-pkg.out -o coverage-pkg.html
	@go tool cover -html=coverage-spawn.out -o coverage-spawn.html
	@go tool cover -html=coverage-truffle.out -o coverage-truffle.html
	@echo "✅ Coverage reports generated:"
	@echo "  coverage-pkg.html     - pkg/i18n coverage"
	@echo "  coverage-spawn.html   - spawn coverage"
	@echo "  coverage-truffle.html - truffle coverage"
	@echo ""
	@echo "Open coverage-*.html files in your browser to view the reports"

# Show help
help:
	@echo "spore-host - The underground network for AWS compute"
	@echo ""
	@echo "Targets:"
	@echo "  make build                  - Build for current platform"
	@echo "  make build-all              - Build for all platforms (Linux, macOS, Windows)"
	@echo "  make clean                  - Clean build artifacts"
	@echo "  make install                - Install to /usr/local/bin (requires sudo)"
	@echo "  make test                   - Run all tests"
	@echo "  make test-i18n              - Run i18n validation tests"
	@echo "  make test-coverage          - Run tests with coverage summary"
	@echo "  make test-coverage-report   - Generate HTML coverage report"
	@echo "  make help                   - Show this help"
	@echo ""
	@echo "Quick start:"
	@echo "  1. make build"
	@echo "  2. ./spawn/bin/spawn"
	@echo ""
	@echo "Testing:"
	@echo "  make test                   - Run all tests"
	@echo "  make test-i18n              - Validate translations (6 languages)"
	@echo "  make test-coverage-report   - View detailed coverage (opens coverage.html)"
	@echo ""
	@echo "Documentation:"
	@echo "  README.md                   - Getting started"
	@echo "  TESTING.md                  - Testing guide"
	@echo "  QUICK_REFERENCE.md          - Command cheat sheet"
	@echo "  COMPLETE_ECOSYSTEM.md       - Full overview"
	@echo "  spawn/MONITORING.md         - spawnd monitoring documentation"

.DEFAULT_GOAL := help
