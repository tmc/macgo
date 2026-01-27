// Package main provides a simple osascript wrapper for managing AppleScripts.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tmc/macgo"
)

func init() {
	// Enable ServicesLauncherV2 by default (uses config file for I/O forwarding with polling)
	// V2 enables I/O forwarding by default (can be disabled with MACGO_DISABLE_IO_FORWARDING=1)
	if os.Getenv("MACGO_SERVICES_VERSION") == "" {
		os.Setenv("MACGO_SERVICES_VERSION", "2")
	}
}

var (
	script     = flag.String("script", "", "name of the script to run")
	list       = flag.Bool("list", false, "list all available scripts")
	create     = flag.String("create", "", "create a new script with the given name")
	edit       = flag.String("edit", "", "edit an existing script")
	remove     = flag.String("remove", "", "remove a script")
	discover   = flag.String("discover", "", "discover AppleScript API for an application")
	generate   = flag.String("generate", "", "generate script using Claude based on discovered API")
	app        = flag.String("app", "", "application name or path for discover/generate")
	prompt     = flag.String("prompt", "", "description of what the script should do")
	scriptsDir = flag.String("dir", "", "scripts directory (default: ~/.osascripts)")
	help       = flag.Bool("help", false, "show help")
)

func main() {
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Request automation permissions
	// The Config struct will use the environment variables set in init()
	// for IO forwarding configuration
	appName := getAppName()
	// If running a script, use the script name as the app name for macgo
	if *script != "" {
		appName = *script
	}
	cfg := &macgo.Config{
		AppName: appName,
		Custom: []string{
			"com.apple.security.automation.apple-events",
		},
		Debug: os.Getenv("MACGO_DEBUG") == "1",
	}

	err := macgo.Start(cfg)
	if err != nil {
		log.Fatalf("Failed to request permissions: %v", err)
	}

	// Get scripts directory
	dir := getScriptsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create scripts directory: %v", err)
	}

	// Handle commands
	switch {
	case *list:
		listScripts(dir)
	case *create != "":
		createScript(dir, *create)
	case *edit != "":
		editScript(dir, *edit)
	case *remove != "":
		removeScript(dir, *remove)
	case *discover != "":
		discoverAPI(*discover)
	case *generate != "":
		generateScript(dir, *generate, *app, *prompt)
	case *script != "":
		runScript(dir, *script)
	default:
		showHelp()
	}
}

func getAppName() string {
	if len(os.Args) > 0 {
		name := filepath.Base(os.Args[0])
		if strings.HasPrefix(name, "go-build") || strings.HasPrefix(name, "__debug_bin") {
			return "osascript-wrapper"
		}
		return name
	}
	return "osascript-wrapper"
}

func getScriptsDir() string {
	if *scriptsDir != "" {
		return *scriptsDir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	return filepath.Join(home, ".osascripts")
}

func listScripts(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("No scripts directory found at %s\n", dir)
		return
	}

	var scripts []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".applescript") {
			name := strings.TrimSuffix(entry.Name(), ".applescript")
			scripts = append(scripts, name)
		}
	}

	if len(scripts) == 0 {
		fmt.Println("No scripts found")
		return
	}

	fmt.Printf("Available scripts in %s:\n", dir)
	for _, script := range scripts {
		fmt.Printf("  %s\n", script)
	}
}

func createScript(dir, name string) {
	scriptPath := filepath.Join(dir, name+".applescript")

	// Check if script already exists
	if _, err := os.Stat(scriptPath); err == nil {
		fmt.Printf("Script '%s' already exists. Use -edit to modify it.\n", name)
		return
	}

	// Create basic template
	template := `-- AppleScript: ` + name + `
-- Edit this script to perform your desired actions

display dialog "Hello from ` + name + `!"
`

	err := os.WriteFile(scriptPath, []byte(template), 0644)
	if err != nil {
		log.Fatalf("Failed to create script: %v", err)
	}

	fmt.Printf("Created script '%s' at %s\n", name, scriptPath)
	fmt.Printf("Edit it with: %s -edit %s\n", getAppName(), name)
}

func editScript(dir, name string) {
	scriptPath := filepath.Join(dir, name+".applescript")

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		fmt.Printf("Script '%s' not found. Use -create to create it.\n", name)
		return
	}

	// Open in editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}

	cmd := exec.Command(editor, scriptPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatalf("Failed to edit script: %v", err)
	}
}

func removeScript(dir, name string) {
	scriptPath := filepath.Join(dir, name+".applescript")

	err := os.Remove(scriptPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Script '%s' not found\n", name)
		} else {
			log.Fatalf("Failed to remove script: %v", err)
		}
		return
	}

	fmt.Printf("Removed script '%s'\n", name)
}

func runScript(dir, name string) {
	scriptPath := filepath.Join(dir, name+".applescript")

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		fmt.Printf("Script '%s' not found. Available scripts:\n", name)
		listScripts(dir)
		return
	}

	fmt.Printf("Running script '%s' from %s...\n", name, scriptPath)

	// Get additional arguments to pass to the script
	scriptArgs := flag.Args()

	// Execute the script with arguments
	cmdArgs := []string{scriptPath}
	cmdArgs = append(cmdArgs, scriptArgs...)

	if os.Getenv("MACGO_DEBUG") == "1" {
		fmt.Printf("Executing: osascript %v\n", cmdArgs)
	}

	cmd := exec.Command("osascript", cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Fatalf("Script failed with exit code %d: %v", exitErr.ExitCode(), err)
		}
		log.Fatalf("Failed to run script: %v", err)
	}

	if os.Getenv("MACGO_DEBUG") == "1" {
		fmt.Printf("Script '%s' completed successfully\n", name)
	}
}

func discoverAPI(appName string) {
	fmt.Printf("Discovering AppleScript API for %s...\n", appName)

	// Run sdef command
	cmd := exec.Command("sdef", appName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Failed to discover API (is the app installed?): %v", err)
	}
}

func generateScript(dir, name, appName, prompt string) {
	if prompt == "" {
		log.Fatal("Must specify -prompt for script generation")
	}
	if appName == "" {
		log.Fatal("Must specify -app for script generation")
	}

	fmt.Printf("Generating script '%s' for %s...\n", name, appName)

	// First, discover the API
	cmd := exec.Command("sdef", appName)
	sdefOutput, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to discover API: %v", err)
	}

	// Prepare prompt for Claude
	claudePrompt := fmt.Sprintf(`You are an AppleScript expert. Generate an AppleScript that does the following:

%s

Here is the AppleScript dictionary (sdef) for %s:

%s

Please generate ONLY the AppleScript code, no explanations. The script should be complete and ready to run.`,
		prompt, appName, string(sdefOutput))

	// Call Claude (use claude command-line tool or API)
	// For now, output the prompt so user can paste it to Claude
	fmt.Println("\n=== Prompt for Claude ===")
	fmt.Println(claudePrompt)
	fmt.Println("\n=== Instructions ===")
	fmt.Println("1. Copy the prompt above")
	fmt.Println("2. Ask Claude to generate the script")
	fmt.Println("3. Save the output with:")
	fmt.Printf("   %s -create %s\n", getAppName(), name)
	fmt.Println("4. Then edit and paste the generated code")

	// TODO: Add direct Claude API integration
	// For now, this serves as a workflow helper
}

func showHelp() {
	fmt.Println("OSAScript Wrapper - Simple AppleScript Management")
	fmt.Println("==============================================")
	fmt.Println()
	fmt.Println("Usage:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	appName := getAppName()
	fmt.Printf("  %s -create my-script                          # Create a new script\n", appName)
	fmt.Printf("  %s -script my-script                          # Run the script\n", appName)
	fmt.Printf("  %s -list                                      # List all scripts\n", appName)
	fmt.Printf("  %s -discover Safari                           # Discover Safari's AppleScript API\n", appName)
	fmt.Printf("  %s -generate my-script -app Safari -prompt 'open google.com'  # Generate script with Claude\n", appName)
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  EDITOR     : Text editor to use for editing scripts (default: nano)")
	fmt.Println()
	fmt.Println("Scripts are stored in ~/.osascripts/ by default")
}