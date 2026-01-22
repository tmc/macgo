// Package main lists application windows with their CGWindowIDs for use with screencapture.
// This tool uses CGo to access Core Graphics APIs and retrieve actual window IDs
// that work with macOS screencapture -l command.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"
)

var (
	app            = flag.String("app", "", "filter by application name (case-insensitive substring match)")
	json           = flag.Bool("json", false, "output as JSON")
	group          = flag.Bool("group", false, "group windows by application")
	all            = flag.Bool("all", false, "show all windows including system UI (Control Center, Window Server, etc.)")
	includeOffscreen = flag.Bool("include-offscreen", false, "include windows that are minimized, hidden, or on different Spaces")
)

// System UI applications to filter out by default
var systemUIApps = map[string]bool{
	"Control Center":      true,
	"Window Server":       true,
	"Notification Center": true,
	"SystemUIServer":      true,
	"Dock":                true,
	"Spotlight":           true,
	"NotificationCenter":  true,
}

// appAliases maps legacy or alternate app names to their canonical names
// This provides compatibility across different macOS versions and common name variations
var appAliases = map[string][]string{
	// System Preferences was renamed to System Settings in macOS Ventura (13.0)
	"System Settings": {"System Preferences", "systemsettings", "systempreferences", "settings", "preferences"},
	"System Preferences": {"System Settings", "systemsettings", "systempreferences", "settings", "preferences"},

	// Common application name variations
	"Activity Monitor": {"activitymonitor"},
	"Disk Utility": {"diskutility"},
	"Terminal": {"terminal.app"},
	"Finder": {"finder"},
	"Safari": {"safari"},
	"Mail": {"mail"},
	"Calendar": {"calendar", "ical"},
	"Contacts": {"contacts", "address book"},
	"Notes": {"notes"},
	"Reminders": {"reminders"},
	"Photos": {"photos"},
	"Music": {"music", "itunes"},
	"Podcasts": {"podcasts"},
	"TV": {"tv", "apple tv"},
	"Books": {"books", "ibooks"},
	"App Store": {"appstore", "app store"},
	"FaceTime": {"facetime"},
	"Messages": {"messages", "imessage"},
}

// resolveAppName attempts to resolve an app name using the alias map
// Returns a slice of possible names to search for (including the original)
func resolveAppName(name string) []string {
	nameLower := strings.ToLower(name)
	names := []string{name} // Always include original name

	// Check if this name is a key in the alias map
	if aliases, ok := appAliases[name]; ok {
		names = append(names, aliases...)
	}

	// Check if this name matches any aliases (case-insensitive)
	for canonical, aliases := range appAliases {
		for _, alias := range aliases {
			if strings.ToLower(alias) == nameLower {
				// Found a match, add the canonical name and all its aliases
				names = append(names, canonical)
				names = append(names, aliases...)
				break
			}
		}
	}

	// Remove duplicates and return
	seen := make(map[string]bool)
	unique := make([]string, 0, len(names))
	for _, n := range names {
		lowerN := strings.ToLower(n)
		if !seen[lowerN] {
			seen[lowerN] = true
			unique = append(unique, n)
		}
	}

	return unique
}

type WindowInfo struct {
	WindowID  int32   `json:"window_id"`
	OwnerPID  int32   `json:"owner_pid"`
	OwnerName string  `json:"owner_name"`
	DisplayID uint32  `json:"display_id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
}

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "list-app-windows - List application windows with CGWindowIDs")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "This tool uses Core Graphics to retrieve actual window IDs that work")
		fmt.Fprintln(os.Stderr, "with macOS screencapture -l command.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "By default, only on-screen windows are shown. System UI windows")
		fmt.Fprintln(os.Stderr, "(Control Center, Window Server, etc.) are filtered out.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Window State Handling:")
		fmt.Fprintln(os.Stderr, "  By default: Only visible windows on current Space")
		fmt.Fprintln(os.Stderr, "  -include-offscreen: Includes minimized/hidden windows and windows on other Spaces")
		fmt.Fprintln(os.Stderr, "  -all: Additionally includes system UI windows")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  list-app-windows [flags]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  # List visible windows (filters out system UI)")
		fmt.Fprintln(os.Stderr, "  list-app-windows")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  # List ALL windows including minimized and hidden")
		fmt.Fprintln(os.Stderr, "  list-app-windows -include-offscreen")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  # List ALL windows including system UI")
		fmt.Fprintln(os.Stderr, "  list-app-windows -all -include-offscreen")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  # Group windows by application")
		fmt.Fprintln(os.Stderr, "  list-app-windows -group")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  # List windows for a specific app (including minimized)")
		fmt.Fprintln(os.Stderr, "  list-app-windows -app Safari -include-offscreen")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  # Get window ID for capturing (even if minimized)")
		fmt.Fprintln(os.Stderr, "  WINDOW_ID=$(list-app-windows -app Safari -include-offscreen | awk 'NR==2 {print $1}')")
		fmt.Fprintln(os.Stderr, "  cd ../screen-capture && ./screen-capture -window $WINDOW_ID -output screenshot.png")
	}
}

func main() {
	flag.Parse()

	// Get window list with appropriate options
	windows, err := getWindowList(*includeOffscreen)
	if err != nil {
		log.Fatal(err)
	}

	// Filter windows
	filtered := make([]WindowInfo, 0)

	// Resolve app name aliases if filtering by app
	var appNames []string
	if *app != "" {
		appNames = resolveAppName(*app)
	}

	for _, w := range windows {
		// Skip system UI unless --all is specified
		if !*all && isSystemUIWindow(w) {
			continue
		}

		// Filter by app name if specified
		if *app != "" {
			found := false
			ownerLower := strings.ToLower(w.OwnerName)

			// Check against all resolved names
			for _, appName := range appNames {
				appLower := strings.ToLower(appName)
				if strings.Contains(ownerLower, appLower) || strings.Contains(appLower, ownerLower) {
					found = true
					break
				}
			}

			if !found {
				continue
			}
		}

		filtered = append(filtered, w)
	}
	windows = filtered

	// Output results
	if *json {
		outputJSON(windows)
	} else if *group {
		outputGrouped(windows)
	} else {
		outputTable(windows)
	}
}

// isSystemUIWindow determines if a window is likely system UI based on heuristics
func isSystemUIWindow(w WindowInfo) bool {
	// Known system UI apps
	if systemUIApps[w.OwnerName] {
		return true
	}

	// Very small windows are likely UI chrome (< 50x50 pixels)
	if w.Width < 50 && w.Height < 50 {
		return true
	}

	return false
}

func outputTable(windows []WindowInfo) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	

	fmt.Fprintln(w, "WINDOW_ID\tDISP\tPID\tOWNER\tWxH")
	for _, win := range windows {
		dims := fmt.Sprintf("%.0fx%.0f", win.Width, win.Height)
		fmt.Fprintf(w, "%d\t%d\t%d\t%s\t%s\n",
			win.WindowID, win.DisplayID, win.OwnerPID, win.OwnerName, dims)
	}
}

func outputJSON(windows []WindowInfo) {
	fmt.Println("[")
	for i, win := range windows {
		comma := ","
		if i == len(windows)-1 {
			comma = ""
		}
		fmt.Printf(`  {"window_id":%d,"display_id":%d,"owner_pid":%d,"owner_name":"%s","x":%.0f,"y":%.0f,"width":%.0f,"height":%.0f}%s`,
			win.WindowID, win.DisplayID, win.OwnerPID, win.OwnerName, win.X, win.Y, win.Width, win.Height, comma)
		fmt.Println()
	}
	fmt.Println("]")
}

func outputGrouped(windows []WindowInfo) {
	// Group windows by application
	appWindows := make(map[string][]WindowInfo)
	for _, win := range windows {
		appWindows[win.OwnerName] = append(appWindows[win.OwnerName], win)
	}

	// Sort app names for consistent output
	appNames := make([]string, 0, len(appWindows))
	for appName := range appWindows {
		appNames = append(appNames, appName)
	}

	// Output grouped by application
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	

	for _, appName := range appNames {
		wins := appWindows[appName]
		plural := ""
		if len(wins) != 1 {
			plural = "s"
		}
		fmt.Fprintf(w, "\n%s (%d window%s):\n", appName, len(wins), plural)
		fmt.Fprintln(w, "  WINDOW_ID\tDISP\tPID\tWxH")
		for _, win := range wins {
			dims := fmt.Sprintf("%.0fx%.0f", win.Width, win.Height)
			fmt.Fprintf(w, "  %d\t%d\t%d\t%s\n",
				win.WindowID, win.DisplayID, win.OwnerPID, dims)
		}
	}
}
