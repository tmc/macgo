package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tmc/macgo"
	"github.com/tmc/macgo/examples/sysprefs-manager/osa"
)

//go:embed applescripts/*.applescript
var embeddedScripts embed.FS

func main() {
	// CLI definitions
	openCmd := flag.NewFlagSet("open", flag.ExitOnError)

	toggleCmd := flag.NewFlagSet("toggle", flag.ExitOnError)
	toggleApp := toggleCmd.String("app", "", "Application name to toggle")
	toggleState := toggleCmd.Bool("enable", true, "Enable (true) or disable (false)")

	usage := func() {
		fmt.Fprintf(os.Stderr, "sysprefs-manager - Automate macOS System Settings\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  sysprefs-manager open <pane-url-or-alias>\n")
		fmt.Fprintf(os.Stderr, "  sysprefs-manager check -app <name>\n")
		fmt.Fprintf(os.Stderr, "  sysprefs-manager toggle -app <name> [-enable=true|false]\n")
		fmt.Fprintf(os.Stderr, "  sysprefs-manager inspect\n")
		fmt.Fprintf(os.Stderr, "  sysprefs-manager close\n")
		fmt.Fprintf(os.Stderr, "\nAliases for open:\n")
		fmt.Fprintf(os.Stderr, "  security, accessibility, screen-recording, automation\n")
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		usage()
	}

	// Initialize macgo (Accessibility is generally required for UI scripting)
	cfg := macgo.NewConfig().
		WithPermissions(macgo.Accessibility).
		FromEnv()

	if err := macgo.Start(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "macgo start failed: %v\n", err)
		os.Exit(1)
	}
	defer macgo.Cleanup()

	// Initialize OSA bridge with embedded scripts
	osa.SetScriptsFS(embeddedScripts)

	switch os.Args[1] {
	case "open":
		openCmd.Parse(os.Args[2:])
		if openCmd.NArg() < 1 {
			fmt.Println("Error: Missing pane argument")
			usage()
		}
		pane := openCmd.Arg(0)
		url := resolvePaneURL(pane)
		fmt.Printf("Opening pane: %s\n", url)

		replacements := map[string]string{
			"{{PANE_URL}}": url,
		}
		if output, err := osa.RunScript("open_pane.applescript", replacements); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Println(output)
		}

	case "toggle":
		toggleCmd.Parse(os.Args[2:])
		if *toggleApp == "" {
			fmt.Println("Error: -app is required")
			usage()
		}
		stateStr := "true"
		if !*toggleState {
			stateStr = "false"
		}
		fmt.Printf("Toggling app '%s' to %s...\n", *toggleApp, stateStr)

		replacements := map[string]string{
			"{{APP_NAME}}":     *toggleApp,
			"{{TARGET_STATE}}": stateStr,
		}
		if output, err := osa.RunScript("toggle_app.applescript", replacements); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Println(output)
		}

	case "check":
		toggleCmd.Parse(os.Args[2:]) // reuse toggle flags for -app
		if *toggleApp == "" {
			fmt.Println("Error: -app is required")
			usage()
		}
		fmt.Printf("Checking permission for app '%s'...\n", *toggleApp)

		replacements := map[string]string{
			"{{APP_NAME}}": *toggleApp,
		}
		if output, err := osa.RunScript("check_app.applescript", replacements); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Println(output)
		}

	case "inspect":
		fmt.Println("Inspecting System Settings UI...")
		if output, err := osa.RunScript("inspect_ui.applescript", nil); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Println(output)
		}

	case "close":
		fmt.Println("Closing System Settings...")
		if output, err := osa.RunScript("close_window.applescript", nil); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Println(output)
		}

	default:
		usage()
	}
}

func resolvePaneURL(alias string) string {
	base := "x-apple.systempreferences:com.apple.preference.security"
	switch strings.ToLower(alias) {
	case "security", "privacy":
		return base
	case "accessibility":
		return base + "?Privacy_Accessibility"
	case "screen-recording", "screen":
		return base + "?Privacy_ScreenCapture"
	case "automation":
		return base + "?Privacy_Automation"
	case "files", "files-folders":
		return base + "?Privacy_FilesAndFolders"
	case "disk", "full-disk":
		return base + "?Privacy_AllFiles"
	}
	// If it looks like a URL, return as is, otherwise default to Security
	if strings.Contains(alias, ":") {
		return alias
	}
	return base
}
