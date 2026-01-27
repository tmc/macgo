// Package main implements a dynamic CLI generator from sdef (AppleScript dictionary) files.
// It parses an application's sdef XML and generates Cobra commands for each AppleScript command.
package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// Sdef structures for parsing AppleScript dictionaries
type Dictionary struct {
	Suites []Suite `xml:"suite"`
}

type Suite struct {
	Name        string    `xml:"name,attr"`
	Code        string    `xml:"code,attr"`
	Description string    `xml:"description,attr"`
	Commands    []Command `xml:"command"`
	Classes     []Class   `xml:"class"`
}

type Command struct {
	Name            string      `xml:"name,attr"`
	Code            string      `xml:"code,attr"`
	Description     string      `xml:"description,attr"`
	DirectParameter Parameter   `xml:"direct-parameter"`
	Parameters      []Parameter `xml:"parameter"`
}

type Parameter struct {
	Name        string `xml:"name,attr"`
	Code        string `xml:"code,attr"`
	Type        string `xml:"type,attr"`
	Description string `xml:"description,attr"`
	Optional    string `xml:"optional,attr"`
}

type Class struct {
	Name        string     `xml:"name,attr"`
	Code        string     `xml:"code,attr"`
	Description string     `xml:"description,attr"`
	Properties  []Property `xml:"property"`
}

type Property struct {
	Name        string `xml:"name,attr"`
	Code        string `xml:"code,attr"`
	Type        string `xml:"type,attr"`
	Description string `xml:"description,attr"`
	Access      string `xml:"access,attr"`
}

var appPath string

func main() {
	// Create root command
	rootCmd := &cobra.Command{
		Use:   "sdef-to-cobra",
		Short: "Dynamic CLI generator from AppleScript dictionaries",
		Long:  "Parses an app's sdef and generates Cobra commands for each AppleScript command",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if appPath == "" {
				log.Fatal("must specify --app")
			}
		},
	}

	rootCmd.PersistentFlags().StringVar(&appPath, "app", "", "Application path (e.g., /Applications/Safari.app)")

	// Add commands that work without loading sdef
	rootCmd.AddCommand(&cobra.Command{
		Use:   "generate",
		Short: "Generate a standalone CLI for an application",
		Run:   func(cmd *cobra.Command, args []string) { generateStandalone() },
	})

	loadCmd := &cobra.Command{
		Use:   "load",
		Short: "Load application's AppleScript API",
		Run:   func(cmd *cobra.Command, args []string) { loadAndExecute(rootCmd) },
	}

	rootCmd.AddCommand(loadCmd)

	// Add test command (doesn't need --app flag)
	testCmd := &cobra.Command{
		Use:               "test",
		Short:             "Run test commands",
		PersistentPreRun:  func(cmd *cobra.Command, args []string) {}, // Override parent pre-run
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Testing AppleScript execution via sdef commands...\n")
			testMusicStatus()
			testSafariOpenURL()
		},
	}
	rootCmd.AddCommand(testCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadAndExecute(rootCmd *cobra.Command) {
	// Get sdef output
	cmd := exec.Command("sdef", appPath)
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("failed to run sdef: %v", err)
	}

	// Parse XML
	var dict Dictionary
	if err := xml.Unmarshal(output, &dict); err != nil {
		log.Fatalf("failed to parse sdef: %v", err)
	}

	appName := strings.TrimSuffix(strings.ToLower(appPath[strings.LastIndex(appPath, "/")+1:]), ".app")
	fmt.Printf("Loaded %s API\n", appName)

	// List available commands
	makeListCommand(dict).Run(nil, nil)
}

func makeCobraCommand(appPath, suiteName string, cmd Command) *cobra.Command {
	// Convert command name to cobra command format
	cmdName := strings.ReplaceAll(cmd.Name, " ", "-")

	cobraCmd := &cobra.Command{
		Use:   cmdName,
		Short: cmd.Description,
		Run: func(cobraCmd *cobra.Command, args []string) {
			executeAppleScript(appPath, cmd, cobraCmd)
		},
	}

	// Add flags for parameters
	for _, param := range cmd.Parameters {
		flagName := strings.ReplaceAll(param.Name, " ", "-")
		defaultVal := ""
		cobraCmd.Flags().String(flagName, defaultVal, param.Description)
	}

	// Add flag for direct parameter
	if cmd.DirectParameter.Type != "" {
		cobraCmd.Flags().String("input", "", cmd.DirectParameter.Description)
	}

	return cobraCmd
}

func executeAppleScript(appPath string, cmd Command, cobraCmd *cobra.Command) {
	// Build AppleScript
	script := fmt.Sprintf("tell application %q\n", appPath)

	cmdName := cmd.Name

	// Get direct parameter if present
	input, _ := cobraCmd.Flags().GetString("input")
	if input != "" {
		script += fmt.Sprintf("\t%s %q", cmdName, input)
	} else {
		script += fmt.Sprintf("\t%s", cmdName)
	}

	// Add parameters
	for _, param := range cmd.Parameters {
		flagName := strings.ReplaceAll(param.Name, " ", "-")
		val, _ := cobraCmd.Flags().GetString(flagName)
		if val != "" {
			script += fmt.Sprintf(" %s %q", param.Name, val)
		}
	}

	script += "\nend tell"

	fmt.Println("Executing AppleScript:")
	fmt.Println(script)
	fmt.Println()

	// Execute
	osacmd := exec.Command("osascript", "-e", script)
	osacmd.Stdout = os.Stdout
	osacmd.Stderr = os.Stderr

	if err := osacmd.Run(); err != nil {
		log.Printf("error: %v", err)
	}
}

func makeListCommand(dict Dictionary) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available commands and classes",
		Run: func(cmd *cobra.Command, args []string) {
			for _, suite := range dict.Suites {
				fmt.Printf("\n%s (%s):\n", suite.Name, suite.Description)

				if len(suite.Commands) > 0 {
					fmt.Println("\nCommands:")
					for _, c := range suite.Commands {
						fmt.Printf("  %-30s %s\n", c.Name, c.Description)
					}
				}

				if len(suite.Classes) > 0 {
					fmt.Println("\nClasses:")
					for _, c := range suite.Classes {
						fmt.Printf("  %-30s %s\n", c.Name, c.Description)
						for _, p := range c.Properties {
							access := "r/w"
							if p.Access == "r" {
								access = "r"
							}
							fmt.Printf("    %-28s %s [%s] (%s)\n",
								p.Name, p.Type, access, p.Description)
						}
					}
				}
			}
		},
	}
}

func makeInspectCommand(appPath string) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect",
		Short: "Show raw sdef output",
		Run: func(cmd *cobra.Command, args []string) {
			osacmd := exec.Command("sdef", appPath)
			osacmd.Stdout = os.Stdout
			osacmd.Stderr = os.Stderr
			osacmd.Run()
		},
	}
}

func generateStandalone() {
	fmt.Println("Standalone generation not yet implemented")
	fmt.Println("This would generate a self-contained CLI tool for the specified application")
}
