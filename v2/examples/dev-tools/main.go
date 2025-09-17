// Package main demonstrates development tools using the v2 macgo API.
// This example shows the simplified configuration for development utilities.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	macgo "github.com/tmc/misc/macgo/v2"
)

// ProjectInfo holds information about a development project
type ProjectInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Language    string    `json:"language"`
	LastOpened  time.Time `json:"last_opened"`
	GitRepo     string    `json:"git_repo,omitempty"`
	HasTests    bool      `json:"has_tests"`
	HasDocker   bool      `json:"has_docker"`
	PackageFile string    `json:"package_file,omitempty"`
}

// BuildResult holds build/test execution results
type BuildResult struct {
	Success  bool          `json:"success"`
	Duration time.Duration `json:"duration"`
	Output   string        `json:"output"`
	Errors   []string      `json:"errors,omitempty"`
}

func main() {
	// Parse command-line flags
	var (
		projectPath = flag.String("path", ".", "Project path to analyze")
		action      = flag.String("action", "analyze", "Action: analyze, build, test, watch, format")
		watch       = flag.Bool("watch", false, "Watch for file changes")
		format      = flag.Bool("format", false, "Auto-format code")
		lint        = flag.Bool("lint", false, "Run linters")
		openInIDE   = flag.String("ide", "", "Open in IDE: vscode, xcode, intellij")
		port        = flag.Int("port", 0, "Start development server on this port")
	)
	flag.Parse()

	// Configure macgo v2 for development tools - much simpler than v1!
	cfg := &macgo.Config{
		AppName:  "DevTools",
		BundleID: "com.example.devtools",
		Permissions: []macgo.Permission{
			macgo.Files,   // Read/write project files
			macgo.Network, // Package downloads, API calls, dev servers
			// Note: v2 automatically includes shell execution capabilities
		},
		LSUIElement: true, // Hide from dock for CLI usage
		Debug:       os.Getenv("MACGO_DEBUG") == "1",
	}

	// Start macgo
	if err := macgo.Start(cfg); err != nil {
		log.Fatalf("Failed to start macgo: %v", err)
	}

	// Resolve project path
	absPath, err := filepath.Abs(*projectPath)
	if err != nil {
		log.Fatalf("Invalid project path: %v", err)
	}

	// Analyze project
	project := analyzeProject(absPath)

	fmt.Printf("🔧 Development Tools (v2 API)\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📁 Project: %s\n", project.Name)
	fmt.Printf("📍 Path: %s\n", project.Path)
	fmt.Printf("🔤 Language: %s\n", project.Language)
	fmt.Printf("🏗 API: v2 (simplified permissions)\n")

	if project.GitRepo != "" {
		fmt.Printf("🔗 Git: %s\n", project.GitRepo)
	}

	fmt.Println()

	// Execute requested action
	switch *action {
	case "analyze":
		showProjectAnalysis(project)

	case "build":
		result := buildProject(project)
		showBuildResult(result)

	case "test":
		result := runTests(project)
		showBuildResult(result)

	case "watch":
		watchProject(project, *format, *lint)

	case "format":
		formatProject(project)

	case "serve":
		if *port == 0 {
			*port = 8080
		}
		startDevServer(project, *port)

	default:
		log.Fatalf("Unknown action: %s", *action)
	}

	// Open in IDE if requested
	if *openInIDE != "" {
		openInIDE(project, *openInIDE)
	}
}

func analyzeProject(path string) *ProjectInfo {
	project := &ProjectInfo{
		Name:       filepath.Base(path),
		Path:       path,
		LastOpened: time.Now(),
	}

	// Detect language and project type
	files, _ := os.ReadDir(path)
	for _, file := range files {
		name := file.Name()

		// Language detection
		switch {
		case name == "go.mod" || name == "go.sum":
			project.Language = "Go"
			project.PackageFile = "go.mod"
		case name == "package.json":
			project.Language = "JavaScript/TypeScript"
			project.PackageFile = "package.json"
		case name == "Cargo.toml":
			project.Language = "Rust"
			project.PackageFile = "Cargo.toml"
		case name == "requirements.txt" || name == "setup.py" || name == "pyproject.toml":
			project.Language = "Python"
			project.PackageFile = name
		case strings.HasSuffix(name, ".xcodeproj") || strings.HasSuffix(name, ".xcworkspace"):
			project.Language = "Swift/Objective-C"
		}

		// Check for other project files
		if name == "Dockerfile" || name == "docker-compose.yml" {
			project.HasDocker = true
		}
		if name == "Makefile" || strings.Contains(name, "test") {
			project.HasTests = true
		}
	}

	// Check for git repository
	if gitInfo := getGitInfo(path); gitInfo != "" {
		project.GitRepo = gitInfo
	}

	return project
}

func showProjectAnalysis(project *ProjectInfo) {
	fmt.Println("📊 Project Analysis")
	fmt.Println("───────────────────")

	// Show project structure
	fmt.Println("\n📂 Structure:")
	showProjectTree(project.Path, "", 0, 3)

	// Show dependencies if package file exists
	if project.PackageFile != "" {
		fmt.Printf("\n📦 Dependencies (%s):\n", project.PackageFile)
		showDependencies(project)
	}

	// Show git status
	if project.GitRepo != "" {
		fmt.Println("\n🔀 Git Status:")
		showGitStatus(project.Path)
	}

	// Show available scripts/tasks
	fmt.Println("\n⚡ Available Tasks:")
	showAvailableTasks(project)

	// Show v2 API benefits
	fmt.Println("\n✨ v2 API Benefits:")
	fmt.Println("  • Single 'Network' permission vs separate client/server")
	fmt.Println("  • Single 'Files' permission vs multiple entitlements")
	fmt.Println("  • No init() functions - explicit configuration")
	fmt.Println("  • Cross-platform safe (no-ops on non-macOS)")
}

func buildProject(project *ProjectInfo) *BuildResult {
	start := time.Now()
	result := &BuildResult{}

	var cmd *exec.Cmd
	switch project.Language {
	case "Go":
		cmd = exec.Command("go", "build", "./...")
	case "JavaScript/TypeScript":
		cmd = exec.Command("npm", "run", "build")
	case "Rust":
		cmd = exec.Command("cargo", "build")
	case "Python":
		// Python doesn't typically have a build step
		result.Success = true
		result.Output = "Python project (no build required) - v2 API"
		return result
	default:
		if _, err := os.Stat(filepath.Join(project.Path, "Makefile")); err == nil {
			cmd = exec.Command("make", "build")
		} else {
			result.Output = "No build system detected - v2 API active"
			return result
		}
	}

	cmd.Dir = project.Path
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(start)
	result.Output = string(output)
	result.Success = err == nil

	if err != nil {
		result.Errors = extractErrors(string(output))
	}

	return result
}

func runTests(project *ProjectInfo) *BuildResult {
	start := time.Now()
	result := &BuildResult{}

	var cmd *exec.Cmd
	switch project.Language {
	case "Go":
		cmd = exec.Command("go", "test", "-v", "./...")
	case "JavaScript/TypeScript":
		cmd = exec.Command("npm", "test")
	case "Rust":
		cmd = exec.Command("cargo", "test")
	case "Python":
		cmd = exec.Command("python", "-m", "pytest", "-v")
	default:
		if _, err := os.Stat(filepath.Join(project.Path, "Makefile")); err == nil {
			cmd = exec.Command("make", "test")
		} else {
			result.Output = "No test runner detected - v2 API"
			return result
		}
	}

	cmd.Dir = project.Path
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(start)
	result.Output = string(output)
	result.Success = err == nil

	if err != nil {
		result.Errors = extractErrors(string(output))
	}

	return result
}

func watchProject(project *ProjectInfo, autoFormat, autoLint bool) {
	fmt.Printf("👁 Watching project for changes (v2 API)...\n")
	fmt.Printf("   Auto-format: %v\n", autoFormat)
	fmt.Printf("   Auto-lint: %v\n", autoLint)
	fmt.Println("\nPress Ctrl+C to stop watching")

	// Simple file watcher simulation
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastCheck := time.Now()
	for range ticker.C {
		// Check for modified files
		modified := findModifiedFiles(project.Path, lastCheck)
		if len(modified) > 0 {
			fmt.Printf("\n🔄 Detected changes in %d file(s)\n", len(modified))

			if autoFormat {
				formatProject(project)
			}

			if autoLint {
				lintProject(project)
			}

			// Run tests if any test files changed
			for _, file := range modified {
				if strings.Contains(file, "test") {
					fmt.Println("🧪 Running tests...")
					result := runTests(project)
					showBuildResult(result)
					break
				}
			}
		}
		lastCheck = time.Now()
	}
}

func formatProject(project *ProjectInfo) {
	fmt.Println("🎨 Formatting code (v2 API)...")

	var cmd *exec.Cmd
	switch project.Language {
	case "Go":
		cmd = exec.Command("go", "fmt", "./...")
	case "JavaScript/TypeScript":
		cmd = exec.Command("npx", "prettier", "--write", ".")
	case "Rust":
		cmd = exec.Command("cargo", "fmt")
	case "Python":
		cmd = exec.Command("black", ".")
	default:
		fmt.Println("No formatter configured for", project.Language)
		return
	}

	cmd.Dir = project.Path
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Format error: %v\n%s", err, output)
	} else {
		fmt.Println("✅ Code formatted successfully")
	}
}

func lintProject(project *ProjectInfo) {
	fmt.Println("🔍 Running linters...")

	var cmd *exec.Cmd
	switch project.Language {
	case "Go":
		cmd = exec.Command("golangci-lint", "run")
	case "JavaScript/TypeScript":
		cmd = exec.Command("npx", "eslint", ".")
	case "Rust":
		cmd = exec.Command("cargo", "clippy")
	case "Python":
		cmd = exec.Command("pylint", ".")
	default:
		fmt.Println("No linter configured for", project.Language)
		return
	}

	cmd.Dir = project.Path
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Lint issues found:\n%s", output)
	} else {
		fmt.Println("✅ No lint issues found")
	}
}

func startDevServer(project *ProjectInfo, port int) {
	fmt.Printf("🚀 Starting development server on port %d (v2 API)...\n", port)

	var cmd *exec.Cmd
	switch project.Language {
	case "Go":
		cmd = exec.Command("go", "run", ".")
	case "JavaScript/TypeScript":
		cmd = exec.Command("npm", "start")
	case "Python":
		cmd = exec.Command("python", "-m", "http.server", fmt.Sprintf("%d", port))
	default:
		fmt.Println("No dev server configuration found")
		return
	}

	cmd.Dir = project.Path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	fmt.Printf("✅ Server running at http://localhost:%d\n", port)
	fmt.Println("Press Ctrl+C to stop")

	cmd.Wait()
}

func openInIDE(project *ProjectInfo, ide string) {
	fmt.Printf("🖥 Opening project in %s (v2 API)...\n", ide)

	var cmd *exec.Cmd
	switch strings.ToLower(ide) {
	case "vscode", "code":
		cmd = exec.Command("code", project.Path)
	case "xcode":
		// Find .xcodeproj or .xcworkspace
		files, _ := os.ReadDir(project.Path)
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".xcworkspace") || strings.HasSuffix(file.Name(), ".xcodeproj") {
				cmd = exec.Command("open", filepath.Join(project.Path, file.Name()))
				break
			}
		}
	case "intellij", "idea":
		cmd = exec.Command("idea", project.Path)
	default:
		fmt.Printf("Unknown IDE: %s\n", ide)
		return
	}

	if cmd != nil {
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to open IDE: %v\n", err)
		}
	}
}

// Helper functions

func getGitInfo(path string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = path
	if output, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(output))
	}
	return ""
}

func showGitStatus(path string) {
	cmd := exec.Command("git", "status", "--short")
	cmd.Dir = path
	if output, err := cmd.Output(); err == nil {
		fmt.Print(string(output))
	}
}

func showProjectTree(path string, indent string, depth, maxDepth int) {
	if depth >= maxDepth {
		return
	}

	entries, _ := os.ReadDir(path)
	for _, entry := range entries {
		// Skip hidden files and common ignore patterns
		if strings.HasPrefix(entry.Name(), ".") || entry.Name() == "node_modules" || entry.Name() == "target" {
			continue
		}

		if entry.IsDir() {
			fmt.Printf("%s📁 %s/\n", indent, entry.Name())
			showProjectTree(filepath.Join(path, entry.Name()), indent+"  ", depth+1, maxDepth)
		} else {
			icon := getFileIcon(entry.Name())
			fmt.Printf("%s%s %s\n", indent, icon, entry.Name())
		}
	}
}

func getFileIcon(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".go":
		return "🔵"
	case ".js", ".ts", ".jsx", ".tsx":
		return "🟨"
	case ".py":
		return "🐍"
	case ".rs":
		return "🦀"
	case ".swift":
		return "🦉"
	case ".json", ".yaml", ".yml":
		return "⚙️"
	case ".md":
		return "📝"
	default:
		return "📄"
	}
}

func showDependencies(project *ProjectInfo) {
	switch project.Language {
	case "Go":
		cmd := exec.Command("go", "list", "-m", "all")
		cmd.Dir = project.Path
		if output, err := cmd.Output(); err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(output))
			count := 0
			for scanner.Scan() && count < 10 {
				line := scanner.Text()
				if !strings.HasPrefix(line, project.Name) {
					fmt.Printf("  • %s\n", line)
					count++
				}
			}
		}

	case "JavaScript/TypeScript":
		packageFile := filepath.Join(project.Path, "package.json")
		if data, err := os.ReadFile(packageFile); err == nil {
			var pkg map[string]interface{}
			if json.Unmarshal(data, &pkg) == nil {
				if deps, ok := pkg["dependencies"].(map[string]interface{}); ok {
					count := 0
					for name, version := range deps {
						fmt.Printf("  • %s@%v\n", name, version)
						count++
						if count >= 10 {
							break
						}
					}
				}
			}
		}
	}
}

func showAvailableTasks(project *ProjectInfo) {
	tasks := []string{}

	switch project.Language {
	case "Go":
		tasks = []string{"go build", "go test", "go fmt", "go vet"}
	case "JavaScript/TypeScript":
		// Read scripts from package.json
		packageFile := filepath.Join(project.Path, "package.json")
		if data, err := os.ReadFile(packageFile); err == nil {
			var pkg map[string]interface{}
			if json.Unmarshal(data, &pkg) == nil {
				if scripts, ok := pkg["scripts"].(map[string]interface{}); ok {
					for name := range scripts {
						tasks = append(tasks, "npm run "+name)
					}
				}
			}
		}
	case "Rust":
		tasks = []string{"cargo build", "cargo test", "cargo fmt", "cargo clippy"}
	case "Python":
		tasks = []string{"python -m pytest", "black .", "pylint ."}
	}

	for _, task := range tasks {
		fmt.Printf("  • %s\n", task)
	}
}

func findModifiedFiles(path string, since time.Time) []string {
	var modified []string
	filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && info.ModTime().After(since) {
			modified = append(modified, file)
		}
		return nil
	})
	return modified
}

func extractErrors(output string) []string {
	var errors []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "error") || strings.Contains(line, "Error") {
			errors = append(errors, line)
		}
	}
	return errors
}

func showBuildResult(result *BuildResult) {
	if result.Success {
		fmt.Printf("✅ Success (%.2fs) - v2 API\n", result.Duration.Seconds())
	} else {
		fmt.Printf("❌ Failed (%.2fs) - v2 API\n", result.Duration.Seconds())
		if len(result.Errors) > 0 {
			fmt.Println("\n🔴 Errors:")
			for _, err := range result.Errors {
				fmt.Printf("  • %s\n", err)
			}
		}
	}

	if result.Output != "" {
		fmt.Println("\n📋 Output:")
		reader := bufio.NewReader(strings.NewReader(result.Output))
		for i := 0; i < 20; i++ {
			line, err := reader.ReadString('\n')
			if err == io.EOF {
				break
			}
			fmt.Print(line)
		}
	}
}