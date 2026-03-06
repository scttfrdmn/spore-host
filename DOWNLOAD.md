# spore-host v0.1.0 - Download Package

## 📦 What's Included

This package contains the complete **spore-host** suite - tools that make AWS EC2 accessible to everyone.

### The Tools

1. **truffle** - Find instances, check quotas, discover capacity
2. **spawn** - Launch instances with wizard or pipe from truffle
3. **spawnd** - Monitor instances automatically (runs on EC2)

### Package Contents

```
spore-host-v0.1.0/
├── README.md                    ← Start here!
├── QUICK_REFERENCE.md          ← Command cheat sheet
├── COMPLETE_ECOSYSTEM.md       ← Full overview
├── CHANGELOG.md                ← Version history
├── LICENSE                     ← MIT License
├── Makefile                    ← Build all tools
│
├── truffle/                     ← Instance discovery
│   ├── README.md
│   ├── QUOTAS.md
│   ├── QUOTA_INTEGRATION.md
│   ├── IMPLEMENTATION.md
│   ├── Makefile
│   ├── go.mod
│   ├── main.go
│   ├── cmd/                     ← Commands (search, spot, capacity, quotas)
│   ├── pkg/                     ← Core packages
│   └── bindings/                ← Python bindings
│       └── python/
│
└── spawn/                       ← Instance launching
    ├── README.md
    ├── ENHANCEMENTS.md
    ├── FINAL_SUMMARY.md
    ├── ECOSYSTEM.md
    ├── IMPLEMENTATION.md
    ├── CLAUDE_CODE_QUICKSTART.md
    ├── Makefile
    ├── go.mod
    ├── main.go
    ├── cmd/                     ← Commands (root, launch, spawnd)
    ├── pkg/                     ← Core packages
    └── scripts/                 ← Deployment scripts
```

---

## 🚀 Quick Start

### 1. Extract the Archive

**Linux/macOS:**
```bash
tar -xzf spore-host-v0.1.0.tar.gz
cd spore-host
```

**Windows:**
```powershell
# Extract spore-host-v0.1.0.zip
cd spore-host
```

### 2. Build the Tools

**Prerequisites:**
- Go 1.21 or later
- (Optional) Python 3.8+ for Python bindings

**Build:**
```bash
# Build everything
make build

# Or build for all platforms
make build-all
```

**Output:**
- `truffle/bin/truffle` (or `truffle.exe` on Windows)
- `spawn/bin/spawn` (or `spawn.exe` on Windows)
- `spawn/bin/spawnd` (Linux only, runs on EC2)

### 3. Configure AWS (for spawn)

```bash
# Option 1: Environment variables
export AWS_ACCESS_KEY_ID=your_key_here
export AWS_SECRET_ACCESS_KEY=your_secret_here
export AWS_DEFAULT_REGION=us-east-1

# Option 2: AWS CLI
aws configure
```

**Note:** truffle works WITHOUT credentials for most features!

### 4. Try It Out!

**No AWS account needed:**
```bash
./truffle/bin/truffle search m7i.large
./truffle/bin/truffle spot m7i.large
```

**With AWS credentials:**
```bash
# Check your quotas
./truffle/bin/truffle quotas

# Launch an instance (wizard mode)
./spawn/bin/spawn

# Or pipe from truffle
./truffle/bin/truffle search m7i.large | ./spawn/bin/spawn
```

---

## 📚 Documentation

### Getting Started
- **README.md** - Overview and quick start
- **QUICK_REFERENCE.md** - Common commands and patterns

### truffle
- **truffle/README.md** - Full user guide
- **truffle/QUOTAS.md** - Quota checking and management
- **truffle/IMPLEMENTATION.md** - Technical details

### spawn
- **spawn/README.md** - Full user guide
- **spawn/ENHANCEMENTS.md** - S3, Windows, and Wizard features
- **spawn/ECOSYSTEM.md** - Visual architecture overview

### Complete Overview
- **COMPLETE_ECOSYSTEM.md** - Everything together

---

## 🔧 Building from Source

### Prerequisites

```bash
# Check Go version
go version  # Should be 1.21+

# Install dependencies
cd spore-host
go mod download
```

### Build Commands

```bash
# Build for your platform
make build

# Build for all platforms (Linux, macOS, Windows)
make build-all

# Run tests
make test

# Install locally (requires sudo)
sudo make install

# Clean build artifacts
make clean
```

### Platform-Specific Builds

```bash
# Build truffle for all platforms
cd truffle
make build-all

# Output:
# bin/truffle-linux-amd64
# bin/truffle-linux-arm64
# bin/truffle-darwin-amd64
# bin/truffle-darwin-arm64

# Build spawn for all platforms
cd spawn
make build-all

# Output:
# bin/spawn-linux-amd64
# bin/spawn-linux-arm64
# bin/spawn-darwin-amd64
# bin/spawn-darwin-arm64
# bin/spawn-windows-amd64.exe
# bin/spawnd-linux-amd64
# bin/spawnd-linux-arm64
```

---

## 🎯 First Steps

### For Absolute Beginners

1. **Explore without AWS account:**
   ```bash
   ./truffle/bin/truffle search m7i.large
   # Learn about instance types, no account needed!
   ```

2. **Configure AWS credentials:**
   ```bash
   aws configure
   ```

3. **Check quotas:**
   ```bash
   ./truffle/bin/truffle quotas
   ```

4. **Launch your first instance:**
   ```bash
   ./spawn/bin/spawn
   # Interactive wizard, just press Enter 6 times!
   ```

### For Power Users

```bash
# Find cheapest Spot instance and launch it
./truffle/bin/truffle spot "m7i.*" --sort-by-price --pick-first | \
  ./spawn/bin/spawn --spot --ttl 8h --idle-timeout 1h
```

### For ML Engineers

```bash
# Check GPU quota
./truffle/bin/truffle quotas --family P

# Find GPU capacity
./truffle/bin/truffle capacity --gpu-only --check-quotas

# Launch with auto-hibernate
./truffle/bin/truffle capacity --instance-types p5.48xlarge | \
  ./spawn/bin/spawn --ttl 24h --hibernate-on-idle
```

---

## 🌟 Key Features

### No AWS Account Needed (for truffle)
- Search instance types
- Check Spot prices
- View capacity availability
- Learn about AWS without committing

### Quota-Aware
- Check quotas before launching
- Prevent "VcpuLimitExceeded" errors
- Generate quota increase requests
- Multi-region quota comparison

### Beginner-Friendly
- Interactive wizard (6 questions)
- Auto-creates SSH keys
- Cost estimates shown upfront
- Clear error messages

### Windows Native
- Full Windows 11/10 support
- PowerShell and CMD compatible
- Handles Windows paths correctly
- Uses OpenSSH for Windows

### Production-Ready
- Auto-termination (no surprise bills)
- Hibernation (save 99% when idle)
- Live progress display
- Comprehensive error handling

---

## 🆘 Common Issues

### "command not found: truffle"

**Solution:** Use the full path or add to PATH:
```bash
# Full path
./truffle/bin/truffle search ...

# Or add to PATH
export PATH=$PATH:$(pwd)/truffle/bin:$(pwd)/spawn/bin
```

### "cannot load AWS credentials"

**Solution:** Configure AWS credentials:
```bash
aws configure
# Or export environment variables
```

### "VcpuLimitExceeded"

**Solution:** Check quotas first:
```bash
./truffle/bin/truffle quotas
./truffle/bin/truffle search ... --check-quotas
```

### "No SSH key found"

**Solution:** Run spawn in wizard mode (creates key automatically):
```bash
./spawn/bin/spawn
```

---

## 📊 What's Next?

### Install Globally (Optional)

```bash
sudo make install
# Now available as:
truffle search ...
spawn --help
```

### Deploy spawnd to S3 (Optional)

For faster instance launches, deploy spawnd to regional S3 buckets:

```bash
cd spawn
./scripts/deploy-spawnd.sh 0.1.0
```

This creates buckets in 10 regions for fast regional downloads.

### Explore Documentation

- Read QUICK_REFERENCE.md for common patterns
- Check COMPLETE_ECOSYSTEM.md for the full story
- Dive into component READMEs for details

---

## 💬 Getting Help

- **Documentation:** See docs in each directory
- **Quick Reference:** QUICK_REFERENCE.md
- **Examples:** README.md files have lots of examples

---

## 🎉 You're Ready!

```bash
# Search instances (no AWS account needed)
./truffle/bin/truffle search m7i.large

# Launch an instance (with AWS credentials)
./spawn/bin/spawn

# Or pipe them together
./truffle/bin/truffle search m7i.large --check-quotas | \
  ./spawn/bin/spawn --ttl 8h
```

**Making AWS accessible to everyone, one instance at a time.** 🍄✨

---

## 📝 Version Information

- **Version:** 0.1.0
- **Release Date:** December 19, 2025
- **License:** MIT
- **Go Version:** 1.21+
- **Platforms:** Linux (x86_64, ARM64), macOS (Intel, M1/M2), Windows (x86_64)

---

**Welcome to the spore-host network!** 🌱
