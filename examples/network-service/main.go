// Package main demonstrates a network service with proper sandbox configuration.
// This example shows how to create a web server that can handle both client and server connections.
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

	"github.com/tmc/misc/macgo"
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
    <title>Network Service Example</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .status { background: #f0f0f0; padding: 15px; border-radius: 8px; margin: 20px 0; }
        .endpoint { background: white; border: 1px solid #ddd; padding: 10px; margin: 10px 0; border-radius: 4px; }
        .method { display: inline-block; width: 60px; font-weight: bold; }
        .path { color: #0066cc; }
        h1 { color: #333; }
        .sandbox { color: green; font-weight: bold; }
    </style>
</head>
<body>
    <h1>🌐 Network Service Example</h1>
    <div class="status">
        <p>Status: <strong>Running</strong></p>
        <p>Uptime: <strong>{{.Uptime}}</strong></p>
        <p>Sandbox: <span class="sandbox">{{if .Sandbox}}✓ Enabled{{else}}✗ Disabled{{end}}</span></p>
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
        <span class="method">WS</span>
        <span class="path">/ws</span> - WebSocket endpoint
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
  -d '{"message":"Hello, World!"}'

# External connectivity test
curl http://localhost:{{.Port}}/api/external
    </pre>
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

	// Configure macgo
	macgo.SetAppName("NetworkService")
	macgo.SetBundleID("com.example.networkservice")

	// Request network permissions
	macgo.RequestEntitlements(
		macgo.EntAppSandbox,
		macgo.EntNetworkServer, // Allow incoming connections
		macgo.EntNetworkClient, // Allow outgoing connections
	)

	// Configure as background service if requested
	if *background {
		macgo.AddPlistEntry("LSUIElement", true)
		macgo.AddPlistEntry("LSBackgroundOnly", true)
	}

	// Start macgo
	macgo.Start()

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
	mux.HandleFunc("/ws", handleWebSocket)

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

		fmt.Printf("🚀 Network service starting...\n")
		fmt.Printf("   Protocol: %s\n", protocol)
		fmt.Printf("   Port: %d\n", config.Port)
		fmt.Printf("   Sandbox: %v\n", macgo.IsInAppBundle())
		if *background {
			fmt.Printf("   Mode: Background (no dock icon)\n")
		}
		fmt.Printf("\n📡 Server listening on %s://localhost:%d\n", protocol, config.Port)

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
	macgo.Stop()
}

func handleIndex(config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.New("index").Parse(indexHTML))
		data := map[string]interface{}{
			"Port":    config.Port,
			"Uptime":  time.Since(startTime).Round(time.Second).String(),
			"Sandbox": macgo.IsInAppBundle(),
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
		Sandbox:   macgo.IsInAppBundle(),
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
			Healthy: macgo.IsInAppBundle(),
			Message: getSandboxMessage(),
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

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Simplified WebSocket handler (would use gorilla/websocket in production)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "WebSocket endpoint (implement with gorilla/websocket or similar)\n")
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
		next.ServeHTTP(w, r)
	})
}

func getSandboxMessage() string {
	if macgo.IsInAppBundle() {
		return "Running in macOS app bundle with sandbox"
	}
	return "Not sandboxed (running as regular binary)"
}