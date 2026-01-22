// Package main implements a Safari CLI tool with macgo permissions management.
package main

import (
	_ "embed"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tmc/macgo"
)

//go:embed Safari.sdef
var safariSdef []byte

type Dictionary struct {
	Suites []Suite `xml:"suite"`
}

type Suite struct {
	Name     string    `xml:"name,attr"`
	Commands []Command `xml:"command"`
}

type Command struct {
	Name        string      `xml:"name,attr"`
	Description string      `xml:"description,attr"`
	Parameters  []Parameter `xml:"parameter"`
}

type Parameter struct {
	Name        string `xml:"name,attr"`
	Type        string `xml:"type,attr"`
	Description string `xml:"description,attr"`
	Optional    string `xml:"optional,attr"`
}

const appPath = "/Applications/Safari.app"

var permissionsRequested bool

func ensurePermissions() {
	if permissionsRequested {
		return
	}
	permissionsRequested = true

	cfg := &macgo.Config{
		AppName: "safari-cli",
		Custom: []string{
			"com.apple.security.automation.apple-events",
		},
		Debug: os.Getenv("MACGO_DEBUG") == "1",
	}

	if err := macgo.Start(cfg); err != nil {
		log.Printf("Warning: permission request failed: %v", err)
	}
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "safari-cli",
		Short: "Safari automation CLI",
		Long:  "Control Safari via AppleScript with proper permissions management",
	}

	// Parse embedded sdef
	var dict Dictionary
	if err := xml.Unmarshal(safariSdef, &dict); err != nil {
		log.Fatalf("failed to parse sdef: %v", err)
	}

	// Add utility commands
	rootCmd.AddCommand(makeListCommand(dict))
	rootCmd.AddCommand(makeCleanupCommand())
	rootCmd.AddCommand(makeTestCommand())

	// Add common Safari commands manually for better UX
	rootCmd.AddCommand(&cobra.Command{
		Use:   "open [url]",
		Short: "Open a URL in Safari",
		Args:  cobra.ExactArgs(1),
		Run:   func(cmd *cobra.Command, args []string) { openURL(args[0]) },
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "get-url",
		Short: "Get the URL of the front tab",
		Run:   func(cmd *cobra.Command, args []string) { getURL() },
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "get-title",
		Short: "Get the title of the front tab",
		Run:   func(cmd *cobra.Command, args []string) { getTitle() },
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "list-tabs",
		Short: "List all open tabs",
		Run:   func(cmd *cobra.Command, args []string) { listTabs() },
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "js [code]",
		Short: "Execute JavaScript in the front tab",
		Args:  cobra.ExactArgs(1),
		Run:   func(cmd *cobra.Command, args []string) { doJavaScript(args[0]) },
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func execScript(script string) (string, error) {
	ensurePermissions()
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func openURL(url string) {
	script := fmt.Sprintf(`tell application "Safari"
	activate
	make new document
	set URL of front document to %q
end tell`, url)

	fmt.Printf("Opening %s...\n", url)
	if _, err := execScript(script); err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Println("✓ Opened")
}

func getURL() {
	script := `tell application "Safari" to get URL of front document`
	output, err := execScript(script)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Println(output)
}

func getTitle() {
	script := `tell application "Safari" to get name of front document`
	output, err := execScript(script)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Println(output)
}

func listTabs() {
	script := `tell application "Safari"
	set tabList to {}
	repeat with w in windows
		repeat with t in tabs of w
			set end of tabList to (name of t & " - " & URL of t)
		end repeat
	end repeat
	return tabList
end tell`

	output, err := execScript(script)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Parse AppleScript list output
	output = strings.Trim(output, "{}")
	if output == "" {
		fmt.Println("No tabs open")
		return
	}

	tabs := strings.Split(output, ", ")
	for i, tab := range tabs {
		fmt.Printf("%d. %s\n", i+1, tab)
	}
}

func doJavaScript(code string) {
	script := fmt.Sprintf(`tell application "Safari" to do JavaScript %q in front document`, code)
	output, err := execScript(script)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	if output != "" {
		fmt.Println(output)
	}
}

func makeListCommand(dict Dictionary) *cobra.Command {
	return &cobra.Command{
		Use:   "list-api",
		Short: "List Safari's AppleScript API",
		Run: func(cmd *cobra.Command, args []string) {
			for _, suite := range dict.Suites {
				if len(suite.Commands) > 0 {
					fmt.Printf("\n%s:\n", suite.Name)
					for _, c := range suite.Commands {
						fmt.Printf("  %-30s %s\n", c.Name, c.Description)
					}
				}
			}
		},
	}
}

func makeCleanupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cleanup-permissions",
		Short: "Remove Safari automation permissions for this tool",
		Long:  "Resets TCC database entry for safari-cli's access to Safari",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Cleaning up automation permissions...")

			// Reset TCC for this binary
			tccCmd := exec.Command("tccutil", "reset", "AppleEvents", "com.apple.Safari")
			if err := tccCmd.Run(); err != nil {
				fmt.Printf("Note: Manual cleanup may be needed in System Settings > Privacy & Security > Automation\n")
			} else {
				fmt.Println("✓ Permissions reset")
			}
		},
	}
}

func makeTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Run test commands to verify functionality",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Testing Safari automation...")

			fmt.Println("1. Opening test page...")
			openURL("https://example.com")

			fmt.Println("\n2. Getting current URL...")
			getURL()

			fmt.Println("\n3. Getting page title...")
			getTitle()

			fmt.Println("\n4. Executing JavaScript...")
			doJavaScript("document.title")

			fmt.Println("\n✓ All tests completed")
		},
	}
}
