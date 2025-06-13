BINARY_NAME = rabbithole
VERSION = 0.1.1
CONFIG_DIR = $(HOME)/.config/rabbithole

# Default target
all: build man

# Build the binary
build:
	go build -o $(BINARY_NAME) main.go

# Generate man page from markdown
man: rabbithole.1

rabbithole.1: rabbithole.1.md
	pandoc -s -t man rabbithole.1.md -o rabbithole.1

# Install binary, man page, and config (requires sudo only for man page)
install: build man install-config
	@echo "Installing binary to /usr/local/bin/..."
	sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "Installing man page to /usr/local/man/man1/..."
	sudo mkdir -p /usr/local/man/man1
	sudo cp rabbithole.1 /usr/local/man/man1/
	sudo mandb -q
	@echo "Installation complete! Try 'man rabbithole'"

# Install config file to user's config directory
install-config:
	@echo "Installing config to $(CONFIG_DIR)..."
	mkdir -p $(CONFIG_DIR)
	@if [ ! -f $(CONFIG_DIR)/config.json ]; then \
		cp config.json $(CONFIG_DIR)/config.json; \
		echo "✅ Config file installed to $(CONFIG_DIR)/config.json"; \
	else \
		echo "⚠️  Config file already exists at $(CONFIG_DIR)/config.json (not overwriting)"; \
	fi

# Install just the binary (no sudo required)
install-bin: build install-config
	@echo "Installing binary to ~/.local/bin/ (make sure it's in PATH)"
	mkdir -p ~/.local/bin
	cp $(BINARY_NAME) ~/.local/bin/

# Test the man page locally
test-man: man
	man ./rabbithole.1

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME) rabbithole.1

# Development workflow - rebuild everything
dev: clean all

# Install missing dependencies
install-deps:
	@echo "Installing dependencies..."
	sudo apt update
	sudo apt install -y pandoc xsel sxhkd dmenu firefox wmctrl xdotool x11-utils

# Check if required tools are available
check-deps:
	@echo "Checking dependencies..."
	@which go > /dev/null || (echo "❌ go not found" && exit 1)
	@which pandoc > /dev/null || (echo "❌ pandoc not found - install with: sudo apt install pandoc" && exit 1) 
	@which xsel > /dev/null || (echo "❌ xsel not found - install with: sudo apt install xsel" && exit 1)
	@which sxhkd > /dev/null || (echo "⚠️  sxhkd not found - install with: sudo apt install sxhkd")
	@which dmenu > /dev/null || (echo "⚠️  dmenu not found - install with: sudo apt install dmenu")
	@which wmctrl > /dev/null || (echo "⚠️  xmctrl not found - install with: sudo apt install wmctrl")
	@which xdotool > /dev/null || (echo "⚠️  xdotool not found - install with: sudo apt install xdotool")
	@which firefox > /dev/null || (echo "⚠️  firefox not found - install with: sudo apt install firefox")
	@echo "✅ Core dependencies satisfied"

# Show available targets
help:
	@echo "Available targets:"
	@echo "  build      - Build the rabbithole binary"
	@echo "  man        - Generate man page from markdown"
	@echo "  all        - Build binary and man page (default)"
	@echo "  install    - Install binary, man page, and config (requires sudo for man page)"
	@echo "  install-config - Install config file to ~/.config/rabbithole/"
	@echo "  install-bin - Install binary to ~/.local/bin and config (no sudo)"
	@echo "  test-man   - View the generated man page locally"
	@echo "  clean      - Remove build artifacts"
	@echo "  dev        - Clean rebuild (clean + all)"
	@echo "  install-deps - Install required dependencies (requires sudo)"
	@echo "  check-deps - Check if required tools are installed"
	@echo "  help       - Show this help"

.PHONY: all build man install install-config install-bin test-man clean dev install-deps check-deps help