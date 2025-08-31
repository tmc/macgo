package macgo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"
)

// BenchmarkConfigCreation benchmarks configuration creation
func BenchmarkConfigCreation(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		config := NewConfig()
		
		// Add some typical configuration
		config.ApplicationName = "TestApp"
		config.BundleID = "com.test.app"
		config.AddEntitlement(EntAppSandbox)
		config.AddEntitlement(EntCamera)
		config.AddPlistEntry("LSUIElement", false)
		config.AddPlistEntry("NSHighResolutionCapable", true)
		
		// Avoid unused variable warning
		_ = config
	}
}

// BenchmarkConfigMerging benchmarks configuration merging performance
func BenchmarkConfigMerging(b *testing.B) {
	// Create base config
	baseConfig := NewConfig()
	baseConfig.ApplicationName = "BaseApp"
	baseConfig.BundleID = "com.base.app"
	baseConfig.AddEntitlement(EntAppSandbox)
	baseConfig.AddPlistEntry("LSUIElement", true)

	// Create merge config with different sizes
	mergeSizes := []struct {
		name         string
		entitlements int
		plistEntries int
	}{
		{"Small", 5, 5},
		{"Medium", 20, 20},
		{"Large", 100, 100},
	}

	for _, size := range mergeSizes {
		b.Run(size.name, func(b *testing.B) {
			// Create merge config
			mergeConfig := NewConfig()
			mergeConfig.ApplicationName = "MergeApp"
			mergeConfig.BundleID = "com.merge.app"
			
			// Add entitlements
			for i := 0; i < size.entitlements; i++ {
				entitlement := Entitlement(fmt.Sprintf("com.test.entitlement.%d", i))
				mergeConfig.AddEntitlement(entitlement)
			}
			
			// Add plist entries
			for i := 0; i < size.plistEntries; i++ {
				key := fmt.Sprintf("TestKey%d", i)
				value := fmt.Sprintf("TestValue%d", i)
				mergeConfig.AddPlistEntry(key, value)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Save original config
				originalConfig := *DefaultConfig
				
				// Merge configuration
				Configure(mergeConfig)
				
				// Restore original config
				*DefaultConfig = originalConfig
			}
		})
	}
}

// BenchmarkEnvironmentVariableParsing benchmarks environment variable parsing
func BenchmarkEnvironmentVariableParsing(b *testing.B) {
	// Set up test environment variables
	testEnvVars := map[string]string{
		"MACGO_APP_NAME":       "TestApp",
		"MACGO_BUNDLE_ID":      "com.test.app",
		"MACGO_CAMERA":         "1",
		"MACGO_MIC":            "1",
		"MACGO_LOCATION":       "1",
		"MACGO_APP_SANDBOX":    "1",
		"MACGO_NETWORK_CLIENT": "1",
		"MACGO_SHOW_DOCK_ICON": "1",
		"MACGO_KEEP_TEMP":      "1",
		"MACGO_NO_RELAUNCH":    "1",
	}

	// Set environment variables
	for key, value := range testEnvVars {
		os.Setenv(key, value)
	}
	defer func() {
		for key := range testEnvVars {
			os.Unsetenv(key)
		}
	}()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create fresh config
		config := NewConfig()
		
		// Parse environment variables (simulating init() function)
		if name := os.Getenv("MACGO_APP_NAME"); name != "" {
			config.ApplicationName = name
		}
		if id := os.Getenv("MACGO_BUNDLE_ID"); id != "" {
			config.BundleID = id
		}
		if os.Getenv("MACGO_NO_RELAUNCH") == "1" {
			config.Relaunch = false
		}
		if os.Getenv("MACGO_KEEP_TEMP") == "1" {
			config.KeepTemp = true
		}
		if os.Getenv("MACGO_SHOW_DOCK_ICON") == "1" {
			config.AddPlistEntry("LSUIElement", false)
		}
		
		// Parse entitlement environment variables
		envVars := map[string]string{
			"MACGO_CAMERA":         string(EntCamera),
			"MACGO_MIC":            string(EntMicrophone),
			"MACGO_LOCATION":       string(EntLocation),
			"MACGO_APP_SANDBOX":    string(EntAppSandbox),
			"MACGO_NETWORK_CLIENT": string(EntNetworkClient),
		}
		
		for env, entitlement := range envVars {
			if os.Getenv(env) == "1" {
				config.AddEntitlement(Entitlement(entitlement))
			}
		}
		
		// Avoid unused variable warning
		_ = config
	}
}

// BenchmarkPlistGeneration benchmarks plist generation with different entry counts
func BenchmarkPlistGeneration(b *testing.B) {
	entryCounts := []struct {
		name  string
		count int
	}{
		{"Small", 10},
		{"Medium", 100},
		{"Large", 1000},
	}

	for _, entryCount := range entryCounts {
		b.Run(entryCount.name, func(b *testing.B) {
			// Create test data
			plistData := make(map[string]any)
			
			// Add various types of entries
			for i := 0; i < entryCount.count; i++ {
				switch i % 5 {
				case 0:
					plistData[fmt.Sprintf("StringKey%d", i)] = fmt.Sprintf("StringValue%d", i)
				case 1:
					plistData[fmt.Sprintf("BoolKey%d", i)] = i%2 == 0
				case 2:
					plistData[fmt.Sprintf("IntKey%d", i)] = i
				case 3:
					plistData[fmt.Sprintf("FloatKey%d", i)] = float64(i) * 1.5
				case 4:
					plistData[fmt.Sprintf("AnyKey%d", i)] = struct{ Value int }{Value: i}
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				tempFile := fmt.Sprintf("/tmp/macgo-bench-plist-%d.plist", i)
				err := writePlist(tempFile, plistData)
				if err != nil {
					b.Fatalf("Failed to write plist: %v", err)
				}
				os.Remove(tempFile)
			}
		})
	}
}

// BenchmarkEntitlementOperations benchmarks entitlement operations
func BenchmarkEntitlementOperations(b *testing.B) {
	operations := []struct {
		name string
		fn   func(config *Config)
	}{
		{
			name: "SingleEntitlement",
			fn: func(config *Config) {
				config.AddEntitlement(EntCamera)
			},
		},
		{
			name: "MultipleEntitlements",
			fn: func(config *Config) {
				config.RequestEntitlements(EntCamera, EntMicrophone, EntLocation, EntAppSandbox)
			},
		},
		{
			name: "ManyEntitlements",
			fn: func(config *Config) {
				entitlements := []Entitlement{
					EntAppSandbox, EntCamera, EntMicrophone, EntLocation, EntAddressBook,
					EntCalendars, EntPhotos, EntReminders, EntNetworkClient, EntNetworkServer,
					EntUserSelectedReadOnly, EntUserSelectedReadWrite, EntBluetooth, EntUSB,
				}
				config.RequestEntitlements(entitlements...)
			},
		},
	}

	for _, op := range operations {
		b.Run(op.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				config := NewConfig()
				op.fn(config)
			}
		})
	}
}

// BenchmarkJSONEntitlementLoading benchmarks loading entitlements from JSON
func BenchmarkJSONEntitlementLoading(b *testing.B) {
	jsonSizes := []struct {
		name         string
		entitlements int
	}{
		{"Small", 5},
		{"Medium", 25},
		{"Large", 100},
	}

	for _, size := range jsonSizes {
		b.Run(size.name, func(b *testing.B) {
			// Create JSON data
			entitlements := make(map[string]bool)
			for i := 0; i < size.entitlements; i++ {
				key := fmt.Sprintf("com.test.entitlement.%d", i)
				entitlements[key] = i%2 == 0
			}

			jsonData, err := json.Marshal(entitlements)
			if err != nil {
				b.Fatalf("Failed to marshal JSON: %v", err)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				err := LoadEntitlementsFromJSON(jsonData)
				if err != nil {
					b.Fatalf("Failed to load entitlements: %v", err)
				}
			}
		})
	}
}

// BenchmarkConfigurationConcurrency benchmarks concurrent configuration operations
func BenchmarkConfigurationConcurrency(b *testing.B) {
	concurrencyLevels := []int{1, 2, 4, 8, 16}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			var wg sync.WaitGroup
			semaphore := make(chan struct{}, concurrency)

			for i := 0; i < b.N; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					// Perform concurrent configuration operations
					config := NewConfig()
					config.ApplicationName = fmt.Sprintf("ConcurrentApp%d", idx)
					config.BundleID = fmt.Sprintf("com.concurrent.app%d", idx)
					
					// Add entitlements
					config.AddEntitlement(EntAppSandbox)
					config.AddEntitlement(EntCamera)
					config.AddEntitlement(EntMicrophone)
					
					// Add plist entries
					config.AddPlistEntry("LSUIElement", false)
					config.AddPlistEntry("NSHighResolutionCapable", true)
					config.AddPlistEntry(fmt.Sprintf("CustomKey%d", idx), fmt.Sprintf("CustomValue%d", idx))
					
					// Test Configure function with concurrent access
					Configure(config)
				}(i)
			}

			wg.Wait()
		})
	}
}

// BenchmarkConfigurationMemoryUsage benchmarks memory usage during configuration
func BenchmarkConfigurationMemoryUsage(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	for i := 0; i < b.N; i++ {
		config := NewConfig()
		config.ApplicationName = fmt.Sprintf("MemoryTestApp%d", i)
		config.BundleID = fmt.Sprintf("com.memory.test%d", i)
		
		// Add many entitlements
		for j := 0; j < 50; j++ {
			entitlement := Entitlement(fmt.Sprintf("com.test.entitlement.%d.%d", i, j))
			config.AddEntitlement(entitlement)
		}
		
		// Add many plist entries
		for j := 0; j < 50; j++ {
			key := fmt.Sprintf("TestKey%d_%d", i, j)
			value := fmt.Sprintf("TestValue%d_%d", i, j)
			config.AddPlistEntry(key, value)
		}
		
		// Merge configuration
		Configure(config)
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Report memory usage
	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "total-bytes/op")
}

// BenchmarkConfigurationCloning benchmarks configuration cloning/copying
func BenchmarkConfigurationCloning(b *testing.B) {
	// Create a complex config to clone
	sourceConfig := NewConfig()
	sourceConfig.ApplicationName = "SourceApp"
	sourceConfig.BundleID = "com.source.app"
	sourceConfig.Relaunch = true
	sourceConfig.KeepTemp = true
	sourceConfig.AutoSign = true
	sourceConfig.SigningIdentity = "Developer ID Application: Test"
	sourceConfig.CustomDestinationAppPath = "/tmp/test.app"
	
	// Add many entitlements
	for i := 0; i < 20; i++ {
		entitlement := Entitlement(fmt.Sprintf("com.test.entitlement.%d", i))
		sourceConfig.AddEntitlement(entitlement)
	}
	
	// Add many plist entries
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("TestKey%d", i)
		value := fmt.Sprintf("TestValue%d", i)
		sourceConfig.AddPlistEntry(key, value)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Clone configuration (manual deep copy)
		clonedConfig := &Config{
			ApplicationName:           sourceConfig.ApplicationName,
			BundleID:                 sourceConfig.BundleID,
			Relaunch:                 sourceConfig.Relaunch,
			KeepTemp:                 sourceConfig.KeepTemp,
			AutoSign:                 sourceConfig.AutoSign,
			SigningIdentity:          sourceConfig.SigningIdentity,
			CustomDestinationAppPath: sourceConfig.CustomDestinationAppPath,
			AppTemplate:              sourceConfig.AppTemplate,
			Entitlements:             make(map[Entitlement]bool),
			PlistEntries:             make(map[string]any),
		}
		
		// Copy entitlements
		for k, v := range sourceConfig.Entitlements {
			clonedConfig.Entitlements[k] = v
		}
		
		// Copy plist entries
		for k, v := range sourceConfig.PlistEntries {
			clonedConfig.PlistEntries[k] = v
		}
		
		// Avoid unused variable warning
		_ = clonedConfig
	}
}

// BenchmarkConfigurationValidation benchmarks configuration validation
func BenchmarkConfigurationValidation(b *testing.B) {
	validationTypes := []struct {
		name string
		fn   func(config *Config) bool
	}{
		{
			name: "BasicValidation",
			fn: func(config *Config) bool {
				return config.ApplicationName != "" && config.BundleID != ""
			},
		},
		{
			name: "EntitlementValidation",
			fn: func(config *Config) bool {
				if config.Entitlements == nil {
					return false
				}
				// Check for required entitlements
				requiredEntitlements := []Entitlement{EntAppSandbox}
				for _, req := range requiredEntitlements {
					if !config.Entitlements[req] {
						return false
					}
				}
				return true
			},
		},
		{
			name: "ComplexValidation",
			fn: func(config *Config) bool {
				// Multiple validation checks
				if config.ApplicationName == "" || config.BundleID == "" {
					return false
				}
				if config.Entitlements == nil {
					return false
				}
				if config.PlistEntries == nil {
					return false
				}
				// Check for conflicting settings
				if config.Relaunch && config.CustomDestinationAppPath == "" {
					return false
				}
				return true
			},
		},
	}

	for _, validationType := range validationTypes {
		b.Run(validationType.name, func(b *testing.B) {
			// Create test config
			config := NewConfig()
			config.ApplicationName = "TestApp"
			config.BundleID = "com.test.app"
			config.AddEntitlement(EntAppSandbox)
			config.AddEntitlement(EntCamera)
			config.AddPlistEntry("LSUIElement", false)
			config.Relaunch = true
			config.CustomDestinationAppPath = "/tmp/test.app"

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				valid := validationType.fn(config)
				if !valid {
					b.Errorf("Configuration validation failed")
				}
			}
		})
	}
}

// BenchmarkConfigurationWithContext benchmarks configuration operations with context
func BenchmarkConfigurationWithContext(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		
		// Simulate configuration operations with context
		config := NewConfig()
		config.ApplicationName = fmt.Sprintf("ContextApp%d", i)
		config.BundleID = fmt.Sprintf("com.context.app%d", i)
		
		// Add entitlements with context awareness
		select {
		case <-ctx.Done():
			cancel()
			continue
		default:
			config.AddEntitlement(EntAppSandbox)
			config.AddEntitlement(EntCamera)
		}
		
		// Add plist entries with context awareness
		select {
		case <-ctx.Done():
			cancel()
			continue
		default:
			config.AddPlistEntry("LSUIElement", false)
			config.AddPlistEntry("NSHighResolutionCapable", true)
		}
		
		// Configure with context
		select {
		case <-ctx.Done():
			cancel()
			continue
		default:
			Configure(config)
		}
		
		cancel()
	}
}

// BenchmarkEntitlementStringConversion benchmarks entitlement string operations
func BenchmarkEntitlementStringConversion(b *testing.B) {
	testEntitlements := []string{
		string(EntAppSandbox),
		string(EntCamera),
		string(EntMicrophone),
		string(EntLocation),
		string(EntAddressBook),
		string(EntNetworkClient),
		string(EntNetworkServer),
		string(EntUserSelectedReadOnly),
		string(EntUserSelectedReadWrite),
		string(EntBluetooth),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, entStr := range testEntitlements {
			// Convert string to Entitlement
			entitlement := Entitlement(entStr)
			
			// Convert back to string
			converted := string(entitlement)
			
			// Verify conversion (simple check)
			if converted != entStr {
				b.Errorf("String conversion failed: expected %s, got %s", entStr, converted)
			}
		}
	}
}