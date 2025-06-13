package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

type SearchEngine struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Key  string `json:"key"`
}

type Config struct {
	SearchEngines []SearchEngine `json:"search_engines"`
	Interface struct {
		Launcher   string   `json:"launcher"`
		DmenuArgs  []string `json:"dmenu_args"`
	} `json:"interface"`
	Database struct {
		Path string `json:"path"`
	} `json:"database"`
	Behavior struct {
		AutoCopyDelayMs    int    `json:"auto_copy_delay_ms"`
		MaxWindows         int    `json:"max_windows"`
		WindowWidth        int    `json:"window_width"`
		WindowHeight       int    `json:"window_height"`
		FirefoxProfile     string `json:"firefox_profile"`
		SelectionMethod    string `json:"selection_method"`
		SelectionTimeoutMs int    `json:"selection_timeout_ms"`
		LogSelections      bool   `json:"log_selections"`
	} `json:"behavior"`
}

var (
	config Config
	db     *sql.DB
	configPath string  // Track which config file was loaded
)

const (
	appName    = "rabbithole"
	appVersion = "0.1.1"
	defaultAutoDelayMs = 500  // Even longer for reliability
	defaultMaxWindows = 5
	defaultWindowWidth = 650   // Smaller window
	defaultWindowHeight = 900  // Even taller
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Window ID normalization functions
func normalizeWindowID(wid string) string {
	if strings.HasPrefix(wid, "0x") {
		return wid
	}
	// Convert decimal to hex
	if val, err := strconv.Atoi(wid); err == nil {
		return fmt.Sprintf("0x%08x", val)
	}
	return wid
}

func waitForNewFirefoxWindow(beforeWIDs map[string]bool) (string, error) {
	timeout := time.Now().Add(5 * time.Second)
	for time.Now().Before(timeout) {
		out, err := exec.Command("wmctrl", "-l").Output()
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Mozilla Firefox") {
				parts := strings.Fields(line)
				if len(parts) > 0 {
					wid := normalizeWindowID(parts[0])
					if !beforeWIDs[wid] {
						return wid, nil
					}
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return "", fmt.Errorf("timeout waiting for new Firefox window")
}

func getDatabasePath() (string, error) {
	var targetUser string
	
	// If running under sudo, use the original user
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		targetUser = sudoUser
	} else {
		// Normal execution, use current user
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		targetUser = usr.Username
	}
	
	// Look up the target user to get their home directory
	usr, err := user.Lookup(targetUser)
	if err != nil {
		return "", err
	}
	
	dbPath := filepath.Join(usr.HomeDir, ".local", "share", "rabbithole", "searches.db")
	
	// Test if we can create the directory
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		// Fallback to system directory
		systemDir := "/var/lib/rabbithole"
		if err := os.MkdirAll(systemDir, 0755); err != nil {
			return "", fmt.Errorf("cannot create database directory in user home (%s) or system location (%s): %w", dbDir, systemDir, err)
		}
		return filepath.Join(systemDir, "searches.db"), nil
	}
	
	return dbPath, nil
}

func ensureConfigAndDB() error {
	if err := loadConfig(); err != nil {
		return err
	}
	return initDatabase()
}

func saveConfig() error {
	if configPath == "" {
		return fmt.Errorf("no config file path known - config may not have been loaded")
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", configPath, err)
	}
	
	return nil
}

func loadConfig() error {
	// Only look in one place - the standard user config location
	configPath = filepath.Join(os.Getenv("HOME"), ".config", "rabbithole", "config.json")
	
	file, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("can't read config file at %s: %w\nRun 'make install-config' to create it", configPath, err)
	}
	
	if err := json.Unmarshal(file, &config); err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	// Set defaults for any missing values
	if config.Database.Path == "" {
		dbPath, err := getDatabasePath()
		if err != nil {
			return fmt.Errorf("couldn't determine database path: %w", err)
		}
		config.Database.Path = dbPath
	}
	
	if config.Behavior.AutoCopyDelayMs == 0 {
		config.Behavior.AutoCopyDelayMs = defaultAutoDelayMs
	}
	
	if config.Behavior.MaxWindows == 0 {
		config.Behavior.MaxWindows = defaultMaxWindows
	}
	
	if config.Behavior.WindowWidth == 0 {
		config.Behavior.WindowWidth = defaultWindowWidth
	}
	
	if config.Behavior.WindowHeight == 0 {
		config.Behavior.WindowHeight = defaultWindowHeight
	}
	
	if config.Behavior.SelectionMethod == "" {
		config.Behavior.SelectionMethod = "auto"
	}
	
	if config.Behavior.SelectionTimeoutMs == 0 {
		config.Behavior.SelectionTimeoutMs = 1000
	}

	return nil
}


func readXSelection(selectionType string) (string, error) {
	var args []string
	switch selectionType {
	case "primary":
		args = []string{"-p"}
	case "clipboard":
		args = []string{"-c"}
	default:
		return "", fmt.Errorf("invalid selection type: %s", selectionType)
	}
	
	cmd := exec.Command("xsel", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("xsel failed: %w", err)
	}
	
	return string(output), nil
}

func captureSelectionSafely() (string, error) {
	method := config.Behavior.SelectionMethod
	
	switch method {
	case "manual":
		return "", fmt.Errorf("selection method set to manual")
	case "primary":
		return captureFromSelection("primary")
	case "clipboard":
		return captureFromSelection("clipboard")
	case "auto":
		fallthrough
	default:
		// Try PRIMARY selection first (highlighted text)
		if text, err := captureFromSelection("primary"); err == nil {
			return text, nil
		}
		
		// Fallback to CLIPBOARD selection (Ctrl+C'd text)
		if text, err := captureFromSelection("clipboard"); err == nil {
			return text, nil
		}
		
		return "", fmt.Errorf("no text in PRIMARY or CLIPBOARD selections")
	}
}

func captureFromSelection(selectionType string) (string, error) {
	text, err := readXSelection(selectionType)
	if err != nil {
		return "", err
	}
	
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", fmt.Errorf("%s selection is empty", selectionType)
	}
	
	if config.Behavior.LogSelections {
		log.Printf("Auto-captured from %s selection (%d chars): %s...", 
			strings.ToUpper(selectionType), len(trimmed), 
			trimmed[:min(30, len(trimmed))])
	} else {
		log.Printf("Auto-captured from %s selection (%d chars)", 
			strings.ToUpper(selectionType), len(trimmed))
	}
	
	return trimmed, nil
}

func getScreenDimensions() (width, height int) {
	cmd := exec.Command("xdpyinfo")
	output, err := cmd.Output()
	if err != nil {
		return 1920, 1080 // reasonable defaults
	}
	
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "dimensions:") {
			fmt.Sscanf(line, "  dimensions:    %dx%d", &width, &height)
			return
		}
	}
	return 1920, 1080
}

func showSearchMenu(query string) (SearchEngine, string, error) {
	// Build menu options - just show engines, not the query
	var options []string
	engineMap := make(map[string]SearchEngine)
	
	for _, engine := range config.SearchEngines {
		option := fmt.Sprintf("%s: %s", engine.Key, engine.Name)
		options = append(options, option)
		engineMap[engine.Key] = engine  // Use key for mapping, not display string
	}

	// Keep prompt clean and consistent
	prompt := "Search with:"

	// Basic dmenu args - horizontal layout
	dmenuArgs := []string{
		"-i",           // case insensitive
		"-p", prompt,
	}

	// Add any custom args from config
	dmenuArgs = append(dmenuArgs, config.Interface.DmenuArgs...)

	// Launch dmenu
	input := strings.Join(options, "\n")
	cmd := exec.Command("dmenu", dmenuArgs...)
	cmd.Stdin = strings.NewReader(input)
	
	output, err := cmd.Output()
	if err != nil {
		return SearchEngine{}, "", fmt.Errorf("dmenu failed: %w", err)
	}
	
	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return SearchEngine{}, "", fmt.Errorf("no selection made")
	}
	
	// Parse selection - could be "k: Kagi" or just "k" for oneshot
	parts := strings.SplitN(selected, ":", 2)
	key := strings.TrimSpace(parts[0])
	
	engine, exists := engineMap[key]
	if !exists {
		return SearchEngine{}, "", fmt.Errorf("invalid selection: %s", selected)
	}
	
	return engine, selected, nil
}

func openBrowserInSideWindow(searchURL, query string) error {
	encodedQuery := url.QueryEscape(query)
	finalURL := strings.ReplaceAll(searchURL, "%s", encodedQuery)
	
	// Get current Firefox windows before launching
	beforeWIDs := make(map[string]bool)
	out, err := exec.Command("wmctrl", "-l").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Mozilla Firefox") {
				parts := strings.Fields(line)
				if len(parts) > 0 {
					wid := normalizeWindowID(parts[0])
					beforeWIDs[wid] = true
				}
			}
		}
	}
	
	// Build Firefox command (without size hints - they're unreliable)
	firefoxArgs := []string{"--new-window", finalURL}
	
	// Add profile if specified
	if config.Behavior.FirefoxProfile != "" {
		firefoxArgs = append(firefoxArgs[:1], 
			append([]string{"--profile", config.Behavior.FirefoxProfile}, 
				firefoxArgs[1:]...)...)
	}
	
	// Launch Firefox
	cmd := exec.Command("firefox", firefoxArgs...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start firefox (is it installed?): %w", err)
	}
	
	// Wait for new Firefox window to appear
	firefoxWID, err := waitForNewFirefoxWindow(beforeWIDs)
	if err != nil {
		return fmt.Errorf("failed to detect new Firefox window: %w", err)
	}
	
	log.Printf("Detected new Firefox window: %s", firefoxWID)
	
	// Get screen dimensions and calculate position
	screenWidth, _ := getScreenDimensions()
	rightMargin := 120
	topMargin := 80
	xPos := screenWidth - config.Behavior.WindowWidth - rightMargin
	yPos := topMargin
	
	// Un-maximize the window first, then position it
	unMaxCmd := exec.Command("wmctrl", "-i", "-r", firefoxWID, "-b", "remove,maximized_vert,maximized_horz")
	if err := unMaxCmd.Run(); err != nil {
		log.Printf("Failed to un-maximize window %s: %v", firefoxWID, err)
	}
	
	// Small delay to let the un-maximize take effect
	time.Sleep(100 * time.Millisecond)
	
	// Position the window
	wmCmd := exec.Command("wmctrl", "-i", "-r", firefoxWID, "-e", 
		fmt.Sprintf("0,%d,%d,%d,%d", xPos, yPos, config.Behavior.WindowWidth, config.Behavior.WindowHeight))
	if err := wmCmd.Run(); err != nil {
		log.Printf("Failed to position window %s: %v", firefoxWID, err)
	} else {
		log.Printf("Successfully positioned Firefox window at %d,%d with size %dx%d", 
			xPos, yPos, config.Behavior.WindowWidth, config.Behavior.WindowHeight)
	}
	
	// Track this as a research window
	if err := addResearchWindow(firefoxWID); err != nil {
		log.Printf("Warning: couldn't track research window: %v", err)
	}
	
	return nil
}

func addResearchWindow(wid string) error {
	// Store research window ID in database (normalized to hex format)
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	normalizedWID := normalizeWindowID(wid)
	_, err := db.Exec("INSERT OR REPLACE INTO research_windows (window_id, created_at) VALUES (?, ?)", 
		normalizedWID, time.Now())
	return err
}

func getResearchWindows() ([]string, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	rows, err := db.Query("SELECT window_id FROM research_windows")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var wids []string
	for rows.Next() {
		var wid string
		if err := rows.Scan(&wid); err != nil {
			continue
		}
		wids = append(wids, wid)
	}
	return wids, nil
}

func cleanupDeadWindows() error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	// Get all stored window IDs
	rows, err := db.Query("SELECT window_id FROM research_windows")
	if err != nil {
		return err
	}
	defer rows.Close()
	
	var storedWIDs []string
	for rows.Next() {
		var wid string
		if err := rows.Scan(&wid); err != nil {
			continue
		}
		storedWIDs = append(storedWIDs, wid)
	}
	
	// Get current window list to check which ones still exist
	out, err := exec.Command("wmctrl", "-l").Output()
	if err != nil {
		return fmt.Errorf("couldn't get window list: %w", err)
	}
	
	// Extract current window IDs
	currentWIDs := make(map[string]bool)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if line != "" {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				currentWIDs[parts[0]] = true
			}
		}
	}
	
	// Remove dead windows from database
	for _, wid := range storedWIDs {
		if !currentWIDs[wid] {
			if err := removeResearchWindow(wid); err != nil {
				log.Printf("Warning: couldn't remove dead window %s: %v", wid, err)
			} else {
				log.Printf("Cleaned up dead research window: %s", wid)
			}
		}
	}
	
	return nil
}

func removeResearchWindow(wid string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	normalizedWID := normalizeWindowID(wid)
	_, err := db.Exec("DELETE FROM research_windows WHERE window_id = ?", normalizedWID)
	return err
}

func closeActiveResearchWindow() error {
	// Find the currently active window
	out, err := exec.Command("xdotool", "getactivewindow").Output()
	if err != nil {
		log.Printf("Failed to get active window with xdotool: %v", err)
		return fmt.Errorf("couldn't get active window (no active window or xdotool failed): %w", err)
	}
	
	activeWID := strings.TrimSpace(string(out))
	normalizedActiveWID := normalizeWindowID(activeWID)
	
	// Clean up dead windows first
	if err := cleanupDeadWindows(); err != nil {
		log.Printf("Warning: couldn't cleanup dead windows: %v", err)
	}
	
	// Check if active window is tracked as a research window
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM research_windows WHERE window_id = ?", normalizedActiveWID).Scan(&count)
	if err != nil {
		return fmt.Errorf("couldn't check research window status: %w", err)
	}
	
	if count == 0 {
		// Silently fail - this is normal when ESC is pressed on non-research windows
		return nil
	}
	
	// Get window name for logging
	nameOut, _ := exec.Command("xdotool", "getwindowname", activeWID).Output()
	windowName := strings.TrimSpace(string(nameOut))
	
	// Close the window
	cmd := exec.Command("xdotool", "windowclose", activeWID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to close window: %w", err)
	}
	
	// Remove from tracking
	if err := removeResearchWindow(activeWID); err != nil {
		log.Printf("Warning: couldn't remove window from tracking: %v", err)
	}
	
	log.Printf("Closed research window: %s (%s)", normalizedActiveWID, windowName)
	return nil
}

func initLogging() error {
	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("couldn't determine user home directory for logging: %w", err)
	}
	
	logDir := filepath.Join(usr.HomeDir, ".local", "share", "rabbithole")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	
	logFile := filepath.Join(logDir, "rabbithole.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	
	// Set log output to file only (no terminal spam)
	log.SetOutput(file)
	return nil
}

func initDatabase() error {
	dbDir := filepath.Dir(config.Database.Path)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	var err error
	db, err = sql.Open("sqlite", config.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	createSearchesTable := `
	CREATE TABLE IF NOT EXISTS searches (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		query TEXT NOT NULL,
		engine_name TEXT NOT NULL,
		engine_url TEXT NOT NULL,
		trigger_method TEXT NOT NULL DEFAULT 'selection',
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		session_id TEXT DEFAULT ''
	);
	`

	createResearchWindowsTable := `
	CREATE TABLE IF NOT EXISTS research_windows (
		window_id TEXT PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := db.Exec(createSearchesTable); err != nil {
		return fmt.Errorf("failed to create searches table: %w", err)
	}

	if _, err := db.Exec(createResearchWindowsTable); err != nil {
		return fmt.Errorf("failed to create research_windows table: %w", err)
	}

	return nil
}

func logSearch(query, engineName, engineURL, triggerMethod string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Simple session ID based on day
	sessionID := time.Now().Format("2006-01-02")
	
	_, err := db.Exec(
		"INSERT INTO searches (query, engine_name, engine_url, trigger_method, session_id) VALUES (?, ?, ?, ?, ?)",
		query, engineName, engineURL, triggerMethod, sessionID,
	)
	return err
}

func handleSearch(query string, triggerMethod string) error {
	engine, _, err := showSearchMenu(query)
	if err != nil {
		return fmt.Errorf("menu selection failed: %w", err)
	}
	
	if query == "" {
		// Prompt for manual query input with paste support
		dmenuInputArgs := []string{
			"-i",  // case insensitive
			"-p", "Enter search query:",
		}
		// Add any custom args from config for consistency (skip duplicates)
		for _, arg := range config.Interface.DmenuArgs {
			if arg != "-i" && arg != "-p" && arg != "Search with:" {
				dmenuInputArgs = append(dmenuInputArgs, arg)
			}
		}
		
		cmd := exec.Command("dmenu", dmenuInputArgs...)
		cmd.Stdin = strings.NewReader("") // Empty input for manual typing/pasting
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("query input failed: %w", err)
		}
		query = strings.TrimSpace(string(output))
		if query == "" {
			return fmt.Errorf("empty query, aborting")
		}
	}
	
	// Log the search
	if err := logSearch(query, engine.Name, engine.URL, triggerMethod); err != nil {
		log.Printf("Failed to log search: %v", err)
	}
	
	// Open browser in side window
	if err := openBrowserInSideWindow(engine.URL, query); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	return nil
}

func setupSxhkd() error {
	fmt.Println("🔧 Rabbit Hole v0.1.1 - Setup")
	fmt.Println("=============================")
	
	// Check dependencies
	deps := []string{"sxhkd", "xdotool", "wmctrl", "xdpyinfo"}
	missing := []string{}
	
	for _, dep := range deps {
		cmd := exec.Command("which", dep)
		if err := cmd.Run(); err != nil {
			missing = append(missing, dep)
		}
	}
	
	if len(missing) > 0 {
		fmt.Println("❌ Missing dependencies:")
		fmt.Printf("   sudo apt install %s\n", strings.Join(missing, " "))
		return fmt.Errorf("missing dependencies: %v", missing)
	}
	
	// Get executable path
	execPath, err := os.Executable()
	if err != nil {
		execPath = "rabbithole"  // Assume it's in PATH
	}
	
	// Create sxhkd config
	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("couldn't determine user home directory for sxhkd setup: %w", err)
	}
	
	configDir := filepath.Join(usr.HomeDir, ".config", "sxhkd")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create sxhkd config directory: %w", err)
	}
	
	configPath := filepath.Join(configDir, "sxhkdrc")
	configContent := fmt.Sprintf(`# Rabbit Hole Investigator hotkeys
ctrl + space
    %s search

ctrl + shift + space
    %s search --empty

# Close active research window
Escape
    %s close
`, execPath, execPath, execPath)
	
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write sxhkd config: %w", err)
	}
	
	fmt.Printf("✅ Created sxhkd config: %s\n", configPath)
	fmt.Println("\n📋 Setup complete! Now:")
	fmt.Println("1. Start sxhkd: sxhkd &")
	fmt.Println("2. Or add to startup (i3: exec sxhkd)")
	fmt.Println("\n⌨️  Hotkeys:")
	fmt.Println("  Ctrl+Space: Search selected text")
	fmt.Println("  Ctrl+Shift+Space: Manual search")
	fmt.Println("  Escape: Close active research window")
	
	return nil
}

func createRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     appName,
		Version: appVersion,
		Short:   "Rabbit Hole - Fast research tool with auto-copy",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initLogging()
		},
	}

	searchCmd := &cobra.Command{
		Use:   "search",
		Short: "Search with auto-copy or manual input",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Hot-reload config and ensure DB is ready
			if err := ensureConfigAndDB(); err != nil {
				return err
			}
			
			empty, _ := cmd.Flags().GetBool("empty")
			var query string
			var triggerMethod string

			if empty {
				query = ""
				triggerMethod = "manual"
			} else {
				// Try safe selection capture first, fall back to manual entry
				var err error
				query, err = captureSelectionSafely()
				if err != nil {
					log.Printf("Selection capture failed, falling back to manual entry: %v", err)
					query = ""
					triggerMethod = "manual"
				} else {
					triggerMethod = "selection"
				}
			}

			return handleSearch(query, triggerMethod)
		},
	}
	searchCmd.Flags().BoolP("empty", "e", false, "Start with empty query")

	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up sxhkd hotkeys",
		RunE: func(cmd *cobra.Command, args []string) error {
			return setupSxhkd()
		},
	}

	closeCmd := &cobra.Command{
		Use:   "close",
		Short: "Close the active research window",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Hot-reload config and ensure DB is ready
			if err := ensureConfigAndDB(); err != nil {
				return err
			}
			return closeActiveResearchWindow()
		},
	}

	addEngineCmd := &cobra.Command{
		Use:   "add-engine [name] [url] [key]",
		Short: "Add a new search engine",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Hot-reload config first
			if err := loadConfig(); err != nil {
				return err
			}
			
			name := args[0]
			url := args[1]
			key := args[2]
			
			// Validate inputs
			if len(key) != 1 {
				return fmt.Errorf("key must be a single character, got: %s", key)
			}
			
			if !strings.Contains(url, "%s") {
				return fmt.Errorf("URL must contain %%s placeholder for query substitution")
			}
			
			// Check for duplicate key
			for _, engine := range config.SearchEngines {
				if engine.Key == key {
					return fmt.Errorf("key '%s' already exists for engine '%s'", key, engine.Name)
				}
			}
			
			// Add the new engine
			newEngine := SearchEngine{
				Name: name,
				URL:  url,
				Key:  key,
			}
			config.SearchEngines = append(config.SearchEngines, newEngine)
			
			// Save the config
			if err := saveConfig(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			
			fmt.Printf("✅ Added search engine: %s (%s) -> %s\n", name, key, url)
			return nil
		},
	}

	listEnginesCmd := &cobra.Command{
		Use:   "list-engines",
		Short: "List all configured search engines",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Hot-reload config first
			if err := loadConfig(); err != nil {
				return err
			}
			
			if len(config.SearchEngines) == 0 {
				fmt.Println("No search engines configured.")
				return nil
			}
			
			fmt.Printf("Configured search engines (%d):\n\n", len(config.SearchEngines))
			for _, engine := range config.SearchEngines {
				fmt.Printf("  %s: %s\n", engine.Key, engine.Name)
				fmt.Printf("     %s\n\n", engine.URL)
			}
			return nil
		},
	}

	removeEngineCmd := &cobra.Command{
		Use:   "remove-engine [key]",
		Short: "Remove a search engine by key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Hot-reload config first
			if err := loadConfig(); err != nil {
				return err
			}
			
			key := args[0]
			
			// Find and remove the engine
			found := false
			newEngines := []SearchEngine{}
			var removedEngine SearchEngine
			
			for _, engine := range config.SearchEngines {
				if engine.Key == key {
					found = true
					removedEngine = engine
				} else {
					newEngines = append(newEngines, engine)
				}
			}
			
			if !found {
				return fmt.Errorf("no search engine found with key '%s'", key)
			}
			
			config.SearchEngines = newEngines
			
			// Save the config
			if err := saveConfig(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			
			fmt.Printf("✅ Removed search engine: %s (%s)\n", removedEngine.Name, key)
			return nil
		},
	}

	editEngineCmd := &cobra.Command{
		Use:   "edit-engine [key] [name] [url] [new-key]",
		Short: "Edit an existing search engine",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Hot-reload config first
			if err := loadConfig(); err != nil {
				return err
			}
			
			oldKey := args[0]
			newName := args[1]
			newURL := args[2]
			newKey := args[3]
			
			// Validate inputs
			if len(newKey) != 1 {
				return fmt.Errorf("key must be a single character, got: %s", newKey)
			}
			
			if !strings.Contains(newURL, "%s") {
				return fmt.Errorf("URL must contain %%s placeholder for query substitution")
			}
			
			// Find the engine to edit
			found := false
			for i, engine := range config.SearchEngines {
				if engine.Key == oldKey {
					found = true
					
					// Check if new key conflicts with other engines (except current one)
					if newKey != oldKey {
						for _, otherEngine := range config.SearchEngines {
							if otherEngine.Key == newKey && otherEngine.Key != oldKey {
								return fmt.Errorf("key '%s' already exists for engine '%s'", newKey, otherEngine.Name)
							}
						}
					}
					
					// Update the engine
					oldEngine := config.SearchEngines[i]
					config.SearchEngines[i] = SearchEngine{
						Name: newName,
						URL:  newURL,
						Key:  newKey,
					}
					
					// Save the config
					if err := saveConfig(); err != nil {
						return fmt.Errorf("failed to save config: %w", err)
					}
					
					fmt.Printf("✅ Updated search engine:\n")
					fmt.Printf("   Old: %s (%s) -> %s\n", oldEngine.Name, oldEngine.Key, oldEngine.URL)
					fmt.Printf("   New: %s (%s) -> %s\n", newName, newKey, newURL)
					return nil
				}
			}
			
			if !found {
				return fmt.Errorf("no search engine found with key '%s'", oldKey)
			}
			
			return nil
		},
	}

	debugSelectionsCmd := &cobra.Command{
		Use:   "debug-selections",
		Short: "Show current X11 selections for troubleshooting",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Current X11 selections:")
			fmt.Println("=======================")
			
			// Check PRIMARY selection
			if primary, err := readXSelection("primary"); err == nil && strings.TrimSpace(primary) != "" {
				fmt.Printf("PRIMARY:   '%s' (%d chars)\n", strings.TrimSpace(primary), len(strings.TrimSpace(primary)))
			} else {
				fmt.Printf("PRIMARY:   (empty or error: %v)\n", err)
			}
			
			// Check CLIPBOARD selection  
			if clipboard, err := readXSelection("clipboard"); err == nil && strings.TrimSpace(clipboard) != "" {
				fmt.Printf("CLIPBOARD: '%s' (%d chars)\n", strings.TrimSpace(clipboard), len(strings.TrimSpace(clipboard)))
			} else {
				fmt.Printf("CLIPBOARD: (empty or error: %v)\n", err)
			}
			
			return nil
		},
	}

	rootCmd.AddCommand(searchCmd, setupCmd, closeCmd, addEngineCmd, listEnginesCmd, removeEngineCmd, editEngineCmd, debugSelectionsCmd)
	return rootCmd
}

func main() {
	rootCmd := createRootCmd()
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
