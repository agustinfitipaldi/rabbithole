# Installation Guide

## Quick Install

```bash
# Install dependencies
sudo apt install pandoc

# Build everything
make all

# Install system-wide (requires sudo)
make install
```

## Manual Steps

### 1. Install Dependencies

```bash
# Required for building
sudo apt install pandoc

# Required for rabbithole functionality  
sudo apt install xsel sxhkd dmenu firefox wmctrl xdotool x11-utils
```

### 2. Build

```bash
# Build binary and man page
make all

# Or build individually
make build    # Just the binary
make man      # Just the man page
```

### 3. Install

**Option A: System-wide (recommended)**
```bash
make install
```

This installs:
- Binary to `/usr/local/bin/rabbithole`
- Man page to `/usr/local/man/man1/rabbithole.1`

**Option B: User-only binary**
```bash
make install-bin
```

This installs just the binary to `~/.local/bin/` (make sure it's in your PATH).

### 4. Setup

```bash
# Generate sxhkd config
rabbithole setup

# Start sxhkd  
sxhkd &

# Test man page
man rabbithole
```

## Build Targets

- `make build` - Build the binary
- `make man` - Generate man page 
- `make all` - Build binary and man page (default)
- `make install` - Install both system-wide (requires sudo)
- `make install-bin` - Install binary to ~/.local/bin
- `make test-man` - View man page locally
- `make clean` - Remove build artifacts
- `make check-deps` - Check dependencies
- `make help` - Show available targets

## Verification

After installation, verify everything works:

```bash
# Check binary is accessible
rabbithole --version

# Check man page
man rabbithole

# Test configuration
rabbithole list-engines
```