# Rabbithole

Linux tool that captures text selections system wide, routes them through a configurable set of search engines, and opens the result in a small firefox window for quick rabbithole entry and exit. Made entirely by Claude Code, coordinated and planned by human (me) :)

## Overview

Rabbithole streamlines the distraction workflow by providing instant access to a set of search engines you configure for any selected text on the system or for empty searches on demand. Searches are logged for future development of rabbithole analysis which will include tree visualization of tab flow and ability to save/revisit parts of a hole.

## Demo

[Screencast from 06-12-2025 11:08:55 AM.webm](https://github.com/user-attachments/assets/71d8de0c-9186-4458-9303-d4446b5704ed)

**Key Features:**
- Sub-50ms response time for hotkey activation
- Automatic text selection capture from X11 PRIMARY (no need to copy)
- Configurable search engines with single-key shortcuts
- Dedicated, positioned research windows
- SQLite logging for research pattern analysis
- Hot-reloadable JSON configuration

## Installation

### Pre-built Packages (Recommended)

**Debian/Ubuntu (.deb package):**

1. Download the latest `.deb` file from the [releases page](https://github.com/yourusername/rabbithole/releases)
2. Double-click to install via Software Center, or:
   ```bash
   sudo apt install ./rabbithole_*_amd64.deb
   ```
3. Configure hotkeys:
   ```bash
   rabbithole setup   # Configure hotkeys
   sxhkd &            # Start hotkey daemon
   ```

**Binary Download:**

1. Download the appropriate binary from the [releases page](https://github.com/yourusername/rabbithole/releases)
2. Extract and install:
   ```bash
   tar -xzf rabbithole_*_linux_x86_64.tar.gz
   sudo mv rabbithole /usr/local/bin/
   rabbithole setup   # Configure hotkeys
   sxhkd &            # Start hotkey daemon
   ```

### From Source

**Prerequisites - Install Go:**

```bash
# 1. Download the latest Linux tarball from https://go.dev/dl/
# 2. Extract and install:
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go*.linux-amd64.tar.gz

# Add Go to your PATH (add this to ~/.bashrc or ~/.zshrc)
export PATH=$PATH:/usr/local/go/bin

# Reload your shell or run:
source ~/.bashrc
```

**Build and Install:**

```bash
git clone <repository-url>
cd rabbithole
make install-deps  # Install dependencies
make install       # Build and install
rabbithole setup   # Configure hotkeys
sxhkd &            # Start hotkey daemon
```

## Usage

### Basic Research Workflow

1. **Highlight text** in any application
2. **Press Ctrl+Space** to launch search menu
3. **Select search engine** using keyboard shortcuts
4. **Research in dedicated window** that opens automatically

### Hotkeys

- `Ctrl+Space`: Search with selected text (defaults to manual if nothing selected)
- `Ctrl+Shift+Space`: Force search with manual input (even if text selected)
- `Escape`: Closes window in focus if its been tracked as a research window

### Search Engine Management

```bash
# List configured engines
rabbithole list-engines

# Add new search engine
rabbithole add-engine "Kagi" "https://kagi.com/?q=%s" "d"

# Remove search engine
rabbithole remove-engine "d"

# Edit existing engine
rabbithole edit-engine "k" "Kagi Search" "https://kagi.com/search?q=%s" "k"
```

## Configuration

Configuration is stored in `config.json` and supports hot-reloading:

```json
{
  "search_engines": [
    {
      "name": "Kagi",
      "url": "https://kagi.com/search?q=%s",
      "key": "k"
    },
    {
      "name": "arXiv",
      "url": "https://arxiv.org/search/?query=%s",
      "key": "a"
    }
  ],
  "behavior": {
    "max_windows": 5,
    "window_width": 650,
    "window_height": 900,
    "selection_method": "auto",
    "firefox_profile": ""
  },
  "database": {
    "path": "~/.local/share/rabbithole/searches.db"
  }
}
```

### Selection Methods

- `"auto"`: Try PRIMARY → manual input (default)
- `"primary"`: Only PRIMARY selection → manual input
- `"manual"`: Always prompt for manual input

## Architecture

The tool consists of:
- **sxhkd**: Global hotkey management
- **Go CLI**: Core search routing and window management
- **dmenu**: Interactive search engine selection
- **Firefox**: Dedicated research windows
- **SQLite**: Search logging and session tracking

## Database Schema

All searches are logged to SQLite with the following structure:

**searches table:**
- Search query and engine information
- Trigger method (selection vs manual)
- Timestamp and session tracking

**research_windows table:**
- Active window tracking for management

## System Requirements

- Linux with X11 (Wayland not supported)
- Firefox browser
- Standard X11 utilities (xsel, wmctrl, xdotool, etc.)

## Development Status

**Current Version: 0.1.1**

This is the initial MVP focused on core search routing functionality. Future releases will include:

- rabbithole tree visualization
- saving rabbitholes, sharing, loading
- zotero or similar hookin for easy cite export?
- research pattern analysis tools

## Troubleshooting

### Research windows show expanded vertical tabs

Firefox remembers sidebar and vertical tab state globally across all windows. If your research windows open with expanded vertical tabs taking up precious space in the small research window:

1. In your main Firefox window, collapse the vertical tabs sidebar by clicking the collapse button
2. Close Firefox completely 
3. Reopen Firefox (this saves the collapsed state as the new default)

New research windows will now open with collapsed tabs, giving you more room for actual content.

### Selection capture not working

If text selection capture fails:

```bash
# Check what's currently in your selections
rabbithole debug-selections

# Test different selection methods in config.json
"selection_method": "primary"    # or "clipboard" or "manual"
```

### Research windows appear too narrow with horizontal tabs

Firefox with horizontal tabs renders research windows slightly narrower than with vertical tabs. For optimal window sizing, enable vertical tabs in Firefox. 

If you prefer horizontal tabs and the narrow window bothers you, the window dimensions should be adjustable in `config.json` (`window_width`/`window_height`) or contributions are welcome to auto-detect tab layout and adjust accordingly.

### Research windows not positioning correctly

Window positioning requires `wmctrl`. Install if missing:

```bash
sudo apt install wmctrl
```

## Contributing

Bug reports and feature requests are welcome. Please include system information and reproduction steps.

## License

MIT License - see LICENSE file for details.
