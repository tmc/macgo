// Package main demonstrates a network service using the v2 macgo API.
// This example shows the simplified configuration for network permissions.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	macgo "github.com/tmc/misc/macgo/v2"
)

// Service configuration
type Config struct {
	Port        int
	EnableHTTPS bool
	CertFile    string
	KeyFile     string
}

// API response types
type StatusResponse struct {
	Status    string    `json:"status"`
	Uptime    string    `json:"uptime"`
	Timestamp time.Time `json:"timestamp"`
	Sandbox   bool      `json:"sandbox"`
	Version   string    `json:"version"`
}

type HealthCheck struct {
	Service string `json:"service"`
	Healthy bool   `json:"healthy"`
	Message string `json:"message"`
}

var (
	startTime = time.Now()
	indexHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>Network Service Example (v2)</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .status { background: #f0f0f0; padding: 15px; border-radius: 8px; margin: 20px 0; }
        .endpoint { background: white; border: 1px solid #ddd; padding: 10px; margin: 10px 0; border-radius: 4px; }
        .method { display: inline-block; width: 60px; font-weight: bold; }
        .path { color: #0066cc; }
        h1 { color: #333; }
        .sandbox { color: green; font-weight: bold; }
        .v2-badge { background: #007acc; color: white; padding: 2px 6px; border-radius: 3px; font-size: 12px; }
    </style>
</head>
<body>
    <h1>🌐 Network Service Example <span class="v2-badge">v2 API</span></h1>
    <div class="status">
        <p>Status: <strong>Running</strong></p>
        <p>Uptime: <strong>{{.Uptime}}</strong></p>
        <p>Sandbox: <span class="sandbox">{{if .Sandbox}}✓ Enabled{{else}}✗ Disabled{{end}}</span></p>
        <p>API Version: <strong>v2</strong></p>
    </div>

    <h2>Available Endpoints:</h2>
    <div class="endpoint">
        <span class="method">GET</span>
        <span class="path">/</span> - This page
    </div>
    <div class="endpoint">
        <span class="method">GET</span>
        <span class="path">/api/status</span> - Service status (JSON)
    </div>
    <div class="endpoint">
        <span class="method">GET</span>
        <span class="path">/api/health</span> - Health check (JSON)
    </div>
    <div class="endpoint">
        <span class="method">POST</span>
        <span class="path">/api/echo</span> - Echo service (JSON)
    </div>
    <div class="endpoint">
        <span class="method">GET</span>
        <span class="path">/api/external</span> - Test external connectivity
    </div>
    <div class="endpoint">
        <span class="method">GET</span>
        <span class="path">/api/permissions</span> - Show active permissions
    </div>

    <h2>Test with curl:</h2>
    <pre>
# Status check
curl http://localhost:{{.Port}}/api/status

# Health check
curl http://localhost:{{.Port}}/api/health

# Echo service
curl -X POST http://localhost:{{.Port}}/api/echo \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello, v2 API!"}'

# External connectivity test
curl http://localhost:{{.Port}}/api/external

# Permission info
curl http://localhost:{{.Port}}/api/permissions
    </pre>

    <h2>v2 API Benefits:</h2>
    <ul>
        <li>Simplified permission configuration</li>
        <li>Explicit, readable setup</li>
        <li>No global state or init() functions</li>
        <li>Better error handling</li>
        <li>Cross-platform by design</li>
    </ul>
</body>
</html>
`
)

func main() {
	// Parse flags
	var (
		port       = flag.Int("port", 8080, "Server port")
		enableTLS  = flag.Bool("tls", false, "Enable HTTPS")
		certFile   = flag.String("cert", "", "TLS certificate file")
		keyFile    = flag.String("key", "", "TLS key file")
		background = flag.Bool("background", false, "Run as background service (no dock icon)")
	)
	flag.Parse()

	// Configure macgo v2 - much cleaner than v1!
	cfg := &macgo.Config{
		AppName:  "NetworkService",
		BundleID: "com.example.networkservice",
		Permissions: []macgo.Permission{
			macgo.Network, // Covers both client and server
		},
		LSUIElement: *background, // Hide from dock if background
		Debug:       os.Getenv("MACGO_DEBUG") == "1",
	}

	// Start macgo
	if err := macgo.Start(cfg); err != nil {
		log.Fatalf("Failed to start macgo: %v", err)
	}

	// Create server configuration
	config := &Config{
		Port:        *port,
		EnableHTTPS: *enableTLS,
		CertFile:    *certFile,
		KeyFile:     *keyFile,
	}

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex(config))
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/echo", handleEcho)
	mux.HandleFunc("/api/external", handleExternal)
	mux.HandleFunc("/api/permissions", handlePermissions)

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: logMiddleware(mux),
	}

	// Start server in goroutine
	go func() {
		protocol := "http"
		if config.EnableHTTPS {
			protocol = "https"
		}

		fmt.Printf("🚀 Network service starting (v2 API)...\n")
		fmt.Printf("   Protocol: %s\n", protocol)
		fmt.Printf("   Port: %d\n", config.Port)
		fmt.Printf("   Sandbox: %v\n", macgo.InAppBundle())
		if *background {
			fmt.Printf("   Mode: Background (no dock icon)\n")
		}
		fmt.Printf("\n📡 Server listening on %s://localhost:%d\n", protocol, config.Port)
		fmt.Printf("💡 Try: curl http://localhost:%d/api/status\n", config.Port)

		var err error
		if config.EnableHTTPS {
			err = server.ListenAndServeTLS(config.CertFile, config.KeyFile)
		} else {
			err = server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n⏹ Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	fmt.Println("✅ Server stopped gracefully")
}

func handleIndex(config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.New("index").Parse(indexHTML))
		data := map[string]interface{}{
			"Port":    config.Port,
			"Uptime":  time.Since(startTime).Round(time.Second).String(),
			"Sandbox": macgo.InAppBundle(),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, data)
	}
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	response := StatusResponse{
		Status:    "online",
		Uptime:    time.Since(startTime).Round(time.Second).String(),
		Timestamp: time.Now(),
		Sandbox:   macgo.InAppBundle(),
		Version:   "v2",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	checks := []HealthCheck{
		{
			Service: "main",
			Healthy: true,
			Message: "Service is running",
		},
		{
			Service: "sandbox",
			Healthy: macgo.InAppBundle(),
			Message: getSandboxMessage(),
		},
		{
			Service: "api",
			Healthy: true,
			Message: "v2 API active",
		},
	}

	allHealthy := true
	for _, check := range checks {
		if !check.Healthy {
			allHealthy = false
			break
		}
	}

	if !allHealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checks)
}

func handleEcho(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"echo":      data,
		"timestamp": time.Now(),
		"headers":   r.Header,
		"api":       "v2",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleExternal(w http.ResponseWriter, r *http.Request) {
	// Test external connectivity
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.github.com/meta")

	result := map[string]interface{}{
		"external_connectivity": err == nil,
		"timestamp":            time.Now(),
		"api":                  "v2",
	}

	if err != nil {
		result["error"] = err.Error()
		result["message"] = "Cannot reach external services (may be sandboxed)"
	} else {
		resp.Body.Close()
		result["message"] = "External connectivity working"
		result["status_code"] = resp.StatusCode
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handlePermissions(w http.ResponseWriter, r *http.Request) {
	// Show information about active permissions
	result := map[string]interface{}{
		"api_version": "v2",
		"sandbox":     macgo.InAppBundle(),
		"permissions": map[string]interface{}{
			"network": map[string]interface{}{
				"enabled":     true,
				"description": "Network client and server access",
				"note":        "v2 uses unified 'Network' permission",
			},
		},
		"configuration": map[string]interface{}{
			"explicit":     true,
			"no_globals":   true,
			"cross_platform": true,
			"description":  "v2 API uses explicit configuration without global state",
		},
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(result)
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s - %v", r.Method, r.RequestURI, r.RemoteAddr, time.Since(start))
	})
}

func getSandboxMessage() string {
	if macgo.InAppBundle() {
		return "Running in macOS app bundle with sandbox (v2 API)"
	}
	return "Not sandboxed (running as regular binary)"
}