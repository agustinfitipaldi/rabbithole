# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Rabbit Hole Investigator is a fast, lightweight Linux tool for academic research that captures text selections, routes them through search engines, and tracks exploration patterns in a tree structure. The core objective is enabling management and analysis of research rabbit holes with <50ms response time as a non-negotiable requirement.

## Architecture

**Current Architecture (v0.1+)**: Using sxhkd for user-space hotkey management to eliminate session isolation issues.

The system consists of 6 core components:
1. **sxhkd**: User-space hotkey management (eliminates root/user session barriers)
2. **Go CLI/Daemon**: CLI tool in v0.1, daemon from v0.2+ for performance
3. **dmenu/rofi Interface**: Quick search launcher interface
4. **Browser Extension**: Minimal Firefox extension that reports navigation to local service
5. **Web UI**: D3.js-based tree visualization and session management on localhost:8080
6. **SQLite Database**: Stores all research data with WAL mode for concurrent access

### Architecture Evolution
```
v0.1: sxhkd → Go CLI → dmenu → browser (+ basic SQLite logging)
v0.2+: sxhkd → Go CLI → Unix Socket → Go daemon → dmenu/SQLite
```

### Key Technical Decisions
- **sxhkd** for hotkey management (user-space, no session isolation)
- **Go CLI** starting simple, evolving to daemon for performance
- **SQLite** for data persistence (single-user, easy backup)
- **dmenu/rofi** for instant search interface (<50ms requirement)
- **D3.js** for tree visualization (unmatched for complex trees)
- **HTTP API** for browser extension communication (simple POST requests)
- **Unix sockets** for CLI→daemon communication (v0.2+)

## Data Models

### Core Schema
```sql
Search {
  id, query, timestamp, session_id,
  trigger_method,  -- 'selection' | 'empty' | 'history'
  corrected_query, -- for fixing typos
  deleted         -- soft delete for mistakes
}

Visit {
  id, search_id, url, title, timestamp, duration_seconds,
  referrer_url,   -- for navigation chains
  is_terminal,    -- did research end here?
  tab_id          -- for tracking parallel browsing paths
}

Session {
  id, name, created_at, tags,
  merged_from_ids,  -- track session merges
  notes            -- quick session-level notes
}
```

## Development Phases

**Currently implementing: v0.1 - MVP Search Router with sxhkd**

The project follows a 6-version roadmap with architectural pivot:
- **v0.1**: MVP CLI Search Router with sxhkd hotkeys and basic logging
- **v0.2**: Daemon mode with Unix socket + comprehensive SQLite persistence
- **v0.3**: Browser extension for navigation tracking
- **v0.4**: Web UI with D3.js tree visualization
- **v0.5**: Research session management and organization
- **v0.6**: Enhanced tree analysis and pattern recognition

## Key Performance Requirements

- **<50ms response time** for hotkey summon (non-negotiable)
- v0.1: Direct CLI execution (sufficient for basic use)
- v0.2+: Daemon stays resident in memory for guaranteed <50ms response
- SQLite with WAL mode for concurrent access
- Simple caching of recent searches
- Debounced navigation tracking (3+ seconds before recording page visits)

## Configuration Structure

The system uses JSON configuration for search engines and daemon settings. Hotkeys are managed by sxhkd. Default hotkeys:
- `CTRL+Space`: Summon with selected text (`rabbithole search`)
- `CTRL+SHIFT+Space`: Empty summon for manual query (`rabbithole search --empty`)
- `CTRL+ALT+R`: Toggle research mode (`rabbithole toggle-research-mode`)

## Development Principles

- **Capture everything from day 1**: Log all interactions even if UI doesn't expose it yet
- **Keep the extension dumb**: All logic stays in Go daemon
- **Default to visible**: Show data in UI early, hide later if needed
- **dmenu stays forever**: Maintain quick launcher even after web UI implementation
- **Daily dogfooding**: Use tool for actual research immediately

## Claude Guidance

- Remember, don't run sudo commands, give them to me