package macgo

import (
	"path/filepath"
	"strings"

	"github.com/tmc/macgo/internal/system"
)

func (c *Config) prepare(execPath string) {
	if c == nil {
		return
	}

	if c.AppName == "" {
		c.AppName = inferAppName(execPath)
	}
	if c.BundleID == "" && c.AppName != "" {
		c.BundleID = system.InferBundleID(c.AppName)
	}

	c.applyLocalNetworkDefaults()
	c.applyPermissionUsageDefaults()
}

func inferAppName(execPath string) string {
	if execPath == "" {
		return ""
	}

	name := system.ExtractAppNameFromPath(filepath.Base(execPath))
	if name == "" {
		return ""
	}
	name = system.CleanAppName(name)
	return system.LimitAppNameLength(name, 251)
}

func (c *Config) applyLocalNetworkDefaults() {
	usage := strings.TrimSpace(c.LocalNetworkUsageDescription)
	services := appendUniqueStrings(existingBonjourServices(c.Info), c.BonjourServices...)
	if usage == "" {
		if existing, ok := c.Info["NSLocalNetworkUsageDescription"].(string); ok {
			usage = strings.TrimSpace(existing)
		}
	}

	hasLocalNetwork := usage != "" || len(services) > 0
	if !hasLocalNetwork {
		return
	}

	c.Permissions = appendUniquePermission(c.Permissions, Network)
	if c.Info == nil {
		c.Info = make(map[string]interface{})
	}

	if usage == "" {
		usage = defaultLocalNetworkUsageDescription(c.AppName)
	}
	if usage != "" {
		c.Info["NSLocalNetworkUsageDescription"] = usage
	}

	if len(services) > 0 {
		c.Info["NSBonjourServices"] = services
	}
}

func existingBonjourServices(info map[string]interface{}) []string {
	if len(info) == 0 {
		return nil
	}

	value, ok := info["NSBonjourServices"]
	if !ok {
		return nil
	}

	switch v := value.(type) {
	case []string:
		return append([]string(nil), v...)
	case []interface{}:
		services := make([]string, 0, len(v))
		for _, item := range v {
			text, ok := item.(string)
			if !ok {
				continue
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			services = append(services, text)
		}
		return services
	case string:
		text := strings.TrimSpace(v)
		if text == "" {
			return nil
		}
		return []string{text}
	default:
		return nil
	}
}

func appendUniquePermission(perms []Permission, perm Permission) []Permission {
	for _, existing := range perms {
		if existing == perm {
			return perms
		}
	}
	return append(perms, perm)
}

func appendUniqueStrings(existing []string, extra ...string) []string {
	if len(existing) == 0 && len(extra) == 0 {
		return nil
	}

	result := make([]string, 0, len(existing)+len(extra))
	seen := make(map[string]struct{}, len(existing)+len(extra))
	appendOne := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	for _, value := range existing {
		appendOne(value)
	}
	for _, value := range extra {
		appendOne(value)
	}

	return result
}

func defaultLocalNetworkUsageDescription(appName string) string {
	if strings.TrimSpace(appName) == "" {
		return "This app needs access to the local network to discover and connect to nearby devices."
	}
	return appName + " needs access to the local network to discover and connect to nearby devices."
}

func (c *Config) applyPermissionUsageDefaults() {
	c.applyPermissionUsage(&permissionUsageConfig{
		perm:        Camera,
		key:         "NSCameraUsageDescription",
		description: strings.TrimSpace(c.CameraUsageDescription),
		defaultText: defaultCameraUsageDescription(c.AppName),
	})
	c.applyPermissionUsage(&permissionUsageConfig{
		perm:        Microphone,
		key:         "NSMicrophoneUsageDescription",
		description: strings.TrimSpace(c.MicrophoneUsageDescription),
		defaultText: defaultMicrophoneUsageDescription(c.AppName),
	})
}

type permissionUsageConfig struct {
	perm        Permission
	key         string
	description string
	defaultText string
}

func (c *Config) applyPermissionUsage(cfg *permissionUsageConfig) {
	description := cfg.description
	if description == "" {
		if existing, ok := c.Info[cfg.key].(string); ok {
			description = strings.TrimSpace(existing)
		}
	}

	hasPermission := hasPermission(c.Permissions, cfg.perm)
	if description == "" && !hasPermission {
		return
	}

	c.Permissions = appendUniquePermission(c.Permissions, cfg.perm)
	if description == "" {
		description = cfg.defaultText
	}
	if description != "" {
		if c.Info == nil {
			c.Info = make(map[string]interface{})
		}
		c.Info[cfg.key] = description
	}
}

func hasPermission(perms []Permission, perm Permission) bool {
	for _, existing := range perms {
		if existing == perm {
			return true
		}
	}
	return false
}

func defaultCameraUsageDescription(appName string) string {
	if strings.TrimSpace(appName) == "" {
		return "This app needs camera access."
	}
	return appName + " needs camera access."
}

func defaultMicrophoneUsageDescription(appName string) string {
	if strings.TrimSpace(appName) == "" {
		return "This app needs microphone access."
	}
	return appName + " needs microphone access."
}
