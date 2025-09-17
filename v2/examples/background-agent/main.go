// Package main demonstrates a background service using the v2 macgo API.
// This example shows the simplified background service configuration.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	macgo "github.com/tmc/misc/macgo/v2"
)

// Agent manages the background service
type Agent struct {
	mu            sync.RWMutex
	startTime     time.Time
	eventCount    int
	lastEventTime time.Time
	logFile       *os.File
	config        *Config
}

// Config holds agent configuration
type Config struct {
	LogPath        string        `json:"log_path"`
	CheckInterval  time.Duration `json:"check_interval"`
	EnableFileWatch bool         `json:"enable_file_watch"`
	WatchPaths     []string      `json:"watch_paths"`
	Port           int           `json:"port"`
}

// Event represents something the agent detected
type Event struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Details   any       `json:"details,omitempty"`
}

func main() {
	// Parse flags
	var (
		configFile  = flag.String("config", "", "Configuration file path")
		logDir      = flag.String("logdir", "~/Library/Logs/BackgroundAgent", "Log directory")
		interval    = flag.Duration("interval", 30*time.Second, "Check interval")
		daemon      = flag.Bool("daemon", false, "Run as daemon (detach from terminal)")
		statusFile  = flag.String("status", "", "Write status to this file")
	)
	flag.Parse()

	// Configure macgo v2 for background operation
	cfg := &macgo.Config{
		AppName:  "BackgroundAgent",
		BundleID: "com.example.backgroundagent",
		Permissions: []macgo.Permission{
			macgo.Files,   // Read config and watch files
			macgo.Network, // Send notifications or metrics
		},
		// Background service configuration - much cleaner in v2!
		LSUIElement:      true, // Hide from dock
		LSBackgroundOnly: true, // Background only
		Custom: []string{
			"NSSupportsSuddenTermination:false", // Graceful shutdown
			"RunAtLoad:true",                    // Auto-start if installed as launch agent
			"KeepAlive:true",                    // Keep running
		},
		Debug: os.Getenv("MACGO_DEBUG") == "1",
	}

	// Start macgo
	if err := macgo.Start(cfg); err != nil {
		log.Fatalf("Failed to start macgo: %v", err)
	}

	// Expand log directory path
	logPath := expandPath(*logDir)
	if err := os.MkdirAll(logPath, 0755); err != nil {
		log.Fatalf("Cannot create log directory: %v", err)
	}

	// Load or create configuration
	config := loadConfig(*configFile, &Config{
		LogPath:       logPath,
		CheckInterval: *interval,
		EnableFileWatch: true,
		WatchPaths: []string{
			"~/Documents",
			"~/Desktop",
		},
	})

	// Create agent
	agent := &Agent{
		startTime: time.Now(),
		config:    config,
	}

	// Open log file
	logFile := filepath.Join(config.LogPath, fmt.Sprintf("agent_%s.log", time.Now().Format("20060102")))
	var err error
	agent.logFile, err = os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Cannot open log file: %v", err)
	}
	defer agent.logFile.Close()

	// Log startup
	agent.logEvent(&Event{
		Type:    "startup",
		Message: "Background agent started (v2 API)",
		Details: map[string]interface{}{
			"pid":        os.Getpid(),
			"sandbox":    macgo.InAppBundle(),
			"config":     config,
			"api_version": "v2",
		},
	})

	fmt.Printf("🤖 Background Agent Started (v2 API)\n")
	fmt.Printf("   PID: %d\n", os.Getpid())
	fmt.Printf("   Logs: %s\n", logFile)
	fmt.Printf("   Interval: %v\n", config.CheckInterval)
	fmt.Printf("   API: v2 (simplified configuration)\n")
	if *daemon {
		fmt.Printf("   Mode: Daemon (detached)\n")
	}
	if *statusFile != "" {
		fmt.Printf("   Status: %s\n", *statusFile)
	}

	// Write status file if requested
	if *statusFile != "" {
		go agent.writeStatusFile(*statusFile)
	}

	// Start background tasks
	go agent.runBackgroundTasks()

	// If running as daemon, detach from terminal
	if *daemon {
		fmt.Println("📡 Agent running in background. Check logs for activity.")
		fmt.Println("   To stop: kill -TERM", os.Getpid())
	}

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan

	// Log shutdown
	agent.logEvent(&Event{
		Type:    "shutdown",
		Message: fmt.Sprintf("Agent shutting down (signal: %v)", sig),
		Details: map[string]interface{}{
			"uptime":      time.Since(agent.startTime).String(),
			"event_count": agent.eventCount,
			"api_version": "v2",
		},
	})

	fmt.Printf("\n⏹ Agent shutting down... (received %v)\n", sig)
	fmt.Printf("   Uptime: %v\n", time.Since(agent.startTime).Round(time.Second))
	fmt.Printf("   Events: %d\n", agent.eventCount)
}

func (a *Agent) runBackgroundTasks() {
	ticker := time.NewTicker(a.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.performChecks()
		}
	}
}

func (a *Agent) performChecks() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Simulate various background checks
	checks := []struct {
		name string
		fn   func() *Event
	}{
		{"system", a.checkSystem},
		{"files", a.checkFiles},
		{"network", a.checkNetwork},
		{"api", a.checkAPI},
	}

	for _, check := range checks {
		if event := check.fn(); event != nil {
			a.logEvent(event)
		}
	}
}

func (a *Agent) checkSystem() *Event {
	// Check system resources
	hostname, _ := os.Hostname()

	return &Event{
		Type:    "system_check",
		Message: "System check completed",
		Details: map[string]interface{}{
			"hostname":    hostname,
			"pid":         os.Getpid(),
			"uptime":      time.Since(a.startTime).Round(time.Second).String(),
			"api_version": "v2",
		},
	}
}

func (a *Agent) checkFiles() *Event {
	if !a.config.EnableFileWatch {
		return nil
	}

	// Simulate file watching
	var changes []string
	for _, path := range a.config.WatchPaths {
		expanded := expandPath(path)
		if info, err := os.Stat(expanded); err == nil {
			// Check for recent modifications
			if time.Since(info.ModTime()) < a.config.CheckInterval {
				changes = append(changes, path)
			}
		}
	}

	if len(changes) > 0 {
		return &Event{
			Type:    "file_change",
			Message: fmt.Sprintf("Detected changes in %d path(s)", len(changes)),
			Details: map[string]interface{}{
				"paths": changes,
			},
		}
	}

	return nil
}

func (a *Agent) checkNetwork() *Event {
	// Simulate network connectivity check
	return &Event{
		Type:    "network_check",
		Message: "Network connectivity verified",
		Details: map[string]interface{}{
			"status": "online",
		},
	}
}

func (a *Agent) checkAPI() *Event {
	// Check API status - this is unique to v2
	return &Event{
		Type:    "api_check",
		Message: "v2 API operating normally",
		Details: map[string]interface{}{
			"version":        "v2",
			"configuration":  "explicit",
			"global_state":   false,
			"init_functions": false,
		},
	}
}

func (a *Agent) logEvent(event *Event) {
	a.mu.Lock()
	defer a.mu.Unlock()

	event.Timestamp = time.Now()
	a.eventCount++
	a.lastEventTime = event.Timestamp

	// Write to log file
	data, _ := json.Marshal(event)
	fmt.Fprintf(a.logFile, "%s %s\n", event.Timestamp.Format(time.RFC3339), string(data))
	a.logFile.Sync()

	// Also log important events to stdout if not detached
	if event.Type == "startup" || event.Type == "shutdown" || event.Type == "error" {
		log.Printf("[%s] %s", event.Type, event.Message)
	}
}

func (a *Agent) writeStatusFile(path string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		a.mu.RLock()
		status := map[string]interface{}{
			"pid":            os.Getpid(),
			"uptime":         time.Since(a.startTime).Seconds(),
			"uptime_string":  time.Since(a.startTime).Round(time.Second).String(),
			"event_count":    a.eventCount,
			"last_event":     a.lastEventTime,
			"last_update":    time.Now(),
			"sandbox":        macgo.InAppBundle(),
			"api_version":    "v2",
			"configuration":  "explicit_config_struct",
		}
		a.mu.RUnlock()

		data, _ := json.MarshalIndent(status, "", "  ")
		os.WriteFile(path, data, 0644)
	}
}

func loadConfig(path string, defaults *Config) *Config {
	if path == "" {
		return defaults
	}

	expanded := expandPath(path)
	data, err := os.ReadFile(expanded)
	if err != nil {
		log.Printf("Cannot read config file %s, using defaults: %v", path, err)
		return defaults
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("Cannot parse config file %s, using defaults: %v", path, err)
		return defaults
	}

	// Apply defaults for missing fields
	if config.LogPath == "" {
		config.LogPath = defaults.LogPath
	}
	if config.CheckInterval == 0 {
		config.CheckInterval = defaults.CheckInterval
	}

	return &config
}

func expandPath(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}
	return path
}