% RABBITHOLE(1) Rabbithole 0.1.1
% Agustin Fitipaldi
% January 2025

# NAME

rabbithole - Fast research tool with auto-copy and search engine routing

# SYNOPSIS

**rabbithole** [*GLOBAL-OPTIONS*] *COMMAND* [*COMMAND-OPTIONS*]

**rabbithole** **search** [**--empty**]  
**rabbithole** **add-engine** *NAME* *URL* *KEY*  
**rabbithole** **list-engines**  
**rabbithole** **remove-engine** *KEY*  
**rabbithole** **edit-engine** *OLD-KEY* *NAME* *URL* *NEW-KEY*  
**rabbithole** **setup**  

# DESCRIPTION

**rabbithole** is a fast, lightweight Linux tool for academic research that captures text selections, routes them through configurable search engines, and tracks exploration patterns. It provides <50ms response time through hotkey integration with **sxhkd(1)** and maintains research sessions in dedicated browser windows.

The core workflow is: select text → press hotkey → choose search engine from menu → open in dedicated research window. All searches are logged to SQLite for future analysis and visualization.

# COMMANDS

## search [--empty]

Launch the interactive search menu. By default, attempts to capture selected text from the active window. If **--empty** is specified, starts with an empty query for manual input.

The search process:
1. Captures selected text from PRIMARY or CLIPBOARD selections (unless **--empty**)
2. Shows **dmenu(1)** with available search engines
3. Opens results in a dedicated, positioned Firefox window
4. Logs the search to SQLite database

**Selection capture** uses X11 selections safely:
- **PRIMARY selection**: Text automatically captured when highlighted
- **CLIPBOARD selection**: Text captured after explicit copy (Ctrl+C)
- **Fallback order**: PRIMARY → CLIPBOARD → manual input

## add-engine *NAME* *URL* *KEY*

Add a new search engine to the configuration.

**NAME**
: Display name for the search engine (e.g., "Duck Duck Go")

**URL**
: Search URL template with **%s** placeholder for query substitution  
  (e.g., "https://duckduckgo.com/?q=%s")

**KEY** 
: Single character shortcut key for dmenu selection (e.g., "d")

The configuration is saved immediately and becomes available for searches without rebuilding.

**Example:**
```
rabbithole add-engine "Duck Duck Go" "https://duckduckgo.com/?q=%s" "d"
```

## list-engines

Display all configured search engines with their keys and URLs. Shows the current configuration loaded from **config.json**.

## remove-engine *KEY*

Remove a search engine by its shortcut key. The change is saved immediately to the configuration file.

## edit-engine *OLD-KEY* *NAME* *URL* *NEW-KEY*

Update an existing search engine's properties. All four parameters are required:

**OLD-KEY**
: Current shortcut key of the engine to modify

**NAME**
: New display name  

**URL**
: New URL template (must contain **%s** placeholder)

**NEW-KEY** 
: New shortcut key (can be the same as old key)

## setup

Generate **sxhkd(1)** configuration for rabbithole hotkeys. Creates **~/.config/sxhkd/sxhkdrc** with the following bindings:

- **Ctrl+Space**: Search with selected text
- **Ctrl+Shift+Space**: Search with manual input

After running setup, start **sxhkd** manually or add to your window manager startup.

# CONFIGURATION

Configuration is stored in **config.json** and loaded fresh on each command execution (hot-reload). The file is searched in the following locations:

1. **./config.json** (current directory)
2. **~/.config/rabbithole/config.json**  
3. **/etc/rabbithole/config.json**

## Search Engines

Search engines are defined in the **search_engines** array:

```json
{
  "search_engines": [
    {
      "name": "Kagi",
      "url": "https://kagi.com/search?q=%s", 
      "key": "k"
    }
  ]
}
```

Each engine requires:
- **name**: Display name shown in dmenu
- **url**: Search URL with **%s** placeholder for query  
- **key**: Single character shortcut (must be unique)

## Interface Configuration

```json
{
  "interface": {
    "launcher": "dmenu",
    "dmenu_args": ["-i", "-p", "Search with:"]
  }
}
```

- **launcher**: Menu program to use (only dmenu supported)
- **dmenu_args**: Additional arguments passed to dmenu

## Window Behavior

```json
{
  "behavior": {
    "auto_copy_delay_ms": 75,
    "window_width": 650,
    "window_height": 900,
    "firefox_profile": "",
    "selection_method": "auto",
    "selection_timeout_ms": 1000,
    "log_selections": false
  }
}
```

- **auto_copy_delay_ms**: Legacy setting (no longer used)
- **window_width/height**: Dimensions for research windows
- **firefox_profile**: Optional Firefox profile for isolation
- **selection_method**: Selection capture behavior
  - `"auto"`: Try PRIMARY → CLIPBOARD → manual (default)
  - `"primary"`: Only PRIMARY → manual
  - `"clipboard"`: Only CLIPBOARD → manual  
  - `"manual"`: Always prompt for input
- **selection_timeout_ms**: Timeout for xsel commands
- **log_selections**: Enable detailed selection capture logging

## Database

```json
{
  "database": {
    "path": "~/.local/share/rabbithole/searches.db"
  }
}
```

SQLite database path for search logging. Created automatically if it doesn't exist.

# HOTKEY INTEGRATION

**rabbithole** is designed to work with **sxhkd(1)** for global hotkey support. After running **rabbithole setup**, start sxhkd:

```bash
sxhkd &
```

Or add to your window manager configuration:

**i3wm (~/.config/i3/config):**
```
exec --no-startup-id sxhkd
```

**bspwm (~/.config/bspwm/bspwmrc):**
```
sxhkd &
```

# WINDOW MANAGEMENT

Research windows are automatically positioned on the right side of the screen. Windows are:

- Positioned at calculated coordinates based on screen size
- Given a distinct window class for identification

The tool uses **wmctrl(1)** and **xdotool(1)** for window positioning.

# DATABASE SCHEMA

Search data is stored in SQLite with the following structure:

## searches table
- **id**: Primary key
- **query**: Search query text
- **engine_name**: Name of search engine used
- **engine_url**: URL template of search engine
- **trigger_method**: 'selection' or 'manual'  
- **timestamp**: When search was performed
- **session_id**: Daily session identifier


# FILES

**~/.config/sxhkd/sxhkdrc**
: sxhkd hotkey configuration (created by **setup**)

**config.json**
: Search engine and behavior configuration

**~/.local/share/rabbithole/searches.db**  
: SQLite database for search logging

**~/.local/share/rabbithole/rabbithole.log**
: Application log file

# DEPENDENCIES

- **xsel(1)**: X11 selection reading (required)
- **sxhkd(1)**: Hotkey daemon
- **dmenu(1)**: Interactive menu  
- **firefox(1)**: Web browser for results
- **wmctrl(1)**: Window manipulation
- **xdotool(1)**: X11 automation  
- **xdpyinfo(1)**: Display information

Install on Debian/Ubuntu:
```bash
sudo apt install xsel sxhkd dmenu firefox wmctrl xdotool x11-utils
```

# EXAMPLES

## Basic Setup

```bash
# Generate sxhkd configuration
rabbithole setup

# Start sxhkd 
sxhkd &

# Now use Ctrl+Space to search selected text
```

## Search Engine Management

```bash
# List current engines
rabbithole list-engines

# Add Duck Duck Go
rabbithole add-engine "Duck Duck Go" "https://duckduckgo.com/?q=%s" "d"

# Add arXiv search  
rabbithole add-engine "arXiv" "https://arxiv.org/search/?query=%s" "a"

# Remove an engine
rabbithole remove-engine "d"

# Edit existing engine
rabbithole edit-engine "k" "Kagi Search" "https://kagi.com/search?q=%s" "k"
```

## Research Workflow

**Method 1 (Instant - PRIMARY selection):**
1. **Highlight text** in any application (automatically in PRIMARY selection)
2. **Press Ctrl+Space** (text captured instantly from PRIMARY)
3. **Choose engine** from dmenu (press key like 'k' for Kagi)
4. **Research window opens** positioned on right side

**Method 2 (Traditional - CLIPBOARD selection):**
1. **Highlight text** → **Ctrl+C** (copies to CLIPBOARD selection)
2. **Press Ctrl+Space** (text captured from CLIPBOARD)
3. **Choose engine** and continue...

**Method 3 (Manual):**
1. **Press Ctrl+Shift+Space** (skip auto-capture)
2. **Type/paste query** manually in dmenu prompt
3. **Choose engine** and continue...

# EXIT STATUS

**0**
: Success

**1** 
: General error (configuration, dependencies, etc.)

# VERSION

This manual page documents **rabbithole** version 0.1.1.

# BUGS

- Window positioning may not work correctly on all window managers
- Auto-copy feature disabled due to system interference (manual entry required)
- X11 only - no Wayland support

Report bugs at: <https://github.com/user/rabbithole/issues>

# SEE ALSO

**sxhkd(1)**, **dmenu(1)**, **firefox(1)**, **wmctrl(1)**, **xdotool(1)**

# COPYRIGHT

Copyright 2025 Agustin Fitipaldi. This is free software; see the source for copying conditions.