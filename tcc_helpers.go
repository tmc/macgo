package macgo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tmc/misc/macgo/internal/system"
)

// TCCDatabase represents a connection to the TCC database
type TCCDatabase struct {
	db       *sql.DB
	readOnly bool
}

// TCCEntry represents a single entry in the TCC database
type TCCEntry struct {
	Service    string    `json:"service"`
	Client     string    `json:"client"`
	ClientType int       `json:"client_type"`
	Auth       int       `json:"auth_value"`
	AuthReason int       `json:"auth_reason"`
	AuthVersion int      `json:"auth_version"`
	LastUsed   time.Time `json:"last_used,omitempty"`
	Allowed    bool      `json:"allowed"`
}

// TCCStatus represents the permission status for a service
type TCCStatus struct {
	Service     string `json:"service"`
	Granted     bool   `json:"granted"`
	LastChecked time.Time `json:"last_checked"`
}

// OpenTCCDatabase opens the TCC database for reading
// Requires Full Disk Access permission
func OpenTCCDatabase() (*TCCDatabase, error) {
	return openTCCDatabaseWithMode(true)
}

// OpenTCCDatabaseForWrite opens the TCC database for writing
// Requires root or special entitlements
func OpenTCCDatabaseForWrite() (*TCCDatabase, error) {
	return openTCCDatabaseWithMode(false)
}

func openTCCDatabaseWithMode(readOnly bool) (*TCCDatabase, error) {
	dbPath := "/Library/Application Support/com.apple.TCC/TCC.db"

	// Check if file exists and is readable
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("cannot access TCC database (need Full Disk Access): %w", err)
	}

	mode := "?mode=ro"
	if !readOnly {
		mode = ""
	}

	db, err := sql.Open("sqlite3", dbPath+mode)
	if err != nil {
		return nil, fmt.Errorf("failed to open TCC database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot connect to TCC database (need Full Disk Access): %w", err)
	}

	return &TCCDatabase{
		db:       db,
		readOnly: readOnly,
	}, nil
}

// Close closes the database connection
func (t *TCCDatabase) Close() error {
	if t.db != nil {
		return t.db.Close()
	}
	return nil
}

// GetPermissionStatus checks if a specific service is granted for a client
func (t *TCCDatabase) GetPermissionStatus(service, client string) (*TCCStatus, error) {
	query := `
		SELECT auth_value
		FROM access
		WHERE service = ? AND client = ?
		ORDER BY indirect_object_identifier_type DESC, auth_value DESC
		LIMIT 1
	`

	var authValue int
	err := t.db.QueryRow(query, service, client).Scan(&authValue)

	status := &TCCStatus{
		Service:     service,
		LastChecked: time.Now(),
	}

	if err == sql.ErrNoRows {
		// No entry means not granted
		status.Granted = false
		return status, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to query permission status: %w", err)
	}

	// Auth values: 0=denied, 1=unknown, 2=allowed, 3=limited
	status.Granted = authValue >= 2

	return status, nil
}

// ListAllPermissions lists all TCC entries in the database
func (t *TCCDatabase) ListAllPermissions() ([]TCCEntry, error) {
	query := `
		SELECT service, client, client_type, auth_value, auth_reason, auth_version,
		       COALESCE(last_used, 0) as last_used
		FROM access
		ORDER BY service, client
	`

	rows, err := t.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	defer rows.Close()

	var entries []TCCEntry
	for rows.Next() {
		var entry TCCEntry
		var lastUsedUnix int64

		err := rows.Scan(
			&entry.Service,
			&entry.Client,
			&entry.ClientType,
			&entry.Auth,
			&entry.AuthReason,
			&entry.AuthVersion,
			&lastUsedUnix,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if lastUsedUnix > 0 {
			entry.LastUsed = time.Unix(lastUsedUnix, 0)
		}
		entry.Allowed = entry.Auth >= 2

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// ListPermissionsForClient lists all permissions for a specific client
func (t *TCCDatabase) ListPermissionsForClient(client string) ([]TCCEntry, error) {
	query := `
		SELECT service, client, client_type, auth_value, auth_reason, auth_version,
		       COALESCE(last_used, 0) as last_used
		FROM access
		WHERE client = ? OR client LIKE ?
		ORDER BY service
	`

	clientPattern := fmt.Sprintf("%%%s%%", filepath.Base(client))
	rows, err := t.db.Query(query, client, clientPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions for client: %w", err)
	}
	defer rows.Close()

	var entries []TCCEntry
	for rows.Next() {
		var entry TCCEntry
		var lastUsedUnix int64

		err := rows.Scan(
			&entry.Service,
			&entry.Client,
			&entry.ClientType,
			&entry.Auth,
			&entry.AuthReason,
			&entry.AuthVersion,
			&lastUsedUnix,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if lastUsedUnix > 0 {
			entry.LastUsed = time.Unix(lastUsedUnix, 0)
		}
		entry.Allowed = entry.Auth >= 2

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// GrantPermission grants a permission for a client (requires write access)
func (t *TCCDatabase) GrantPermission(service, client string) error {
	if t.readOnly {
		return fmt.Errorf("database opened in read-only mode")
	}

	query := `
		INSERT OR REPLACE INTO access
		(service, client, client_type, auth_value, auth_reason, auth_version, csreq, last_modified)
		VALUES (?, ?, 0, 2, 3, 1, NULL, CAST(strftime('%s', 'now') AS INTEGER))
	`

	_, err := t.db.Exec(query, service, client)
	if err != nil {
		return fmt.Errorf("failed to grant permission: %w", err)
	}

	return nil
}

// RevokePermission revokes a permission for a client (requires write access)
func (t *TCCDatabase) RevokePermission(service, client string) error {
	if t.readOnly {
		return fmt.Errorf("database opened in read-only mode")
	}

	query := `DELETE FROM access WHERE service = ? AND client = ?`

	_, err := t.db.Exec(query, service, client)
	if err != nil {
		return fmt.Errorf("failed to revoke permission: %w", err)
	}

	return nil
}

// ResetPermissionsForClient removes all permissions for a client (requires write access)
func (t *TCCDatabase) ResetPermissionsForClient(client string) error {
	if t.readOnly {
		return fmt.Errorf("database opened in read-only mode")
	}

	query := `DELETE FROM access WHERE client = ? OR client LIKE ?`
	clientPattern := fmt.Sprintf("%%%s%%", filepath.Base(client))

	_, err := t.db.Exec(query, client, clientPattern)
	if err != nil {
		return fmt.Errorf("failed to reset permissions: %w", err)
	}

	return nil
}

// CheckFullDiskAccess checks if the current process has Full Disk Access
func CheckFullDiskAccess() bool {
	dbPath := "/Library/Application Support/com.apple.TCC/TCC.db"

	// Try to open the file for reading
	file, err := os.Open(dbPath)
	if err != nil {
		return false
	}
	file.Close()

	// Try to actually read from the database
	db, err := OpenTCCDatabase()
	if err != nil {
		return false
	}
	defer db.Close()

	// If we can query it, we have FDA
	_, err = db.GetPermissionStatus("kTCCServiceSystemPolicyAllFiles", "test")
	return err == nil
}

// WaitForFullDiskAccess waits for Full Disk Access to be granted
func WaitForFullDiskAccess(ctx context.Context, checkInterval time.Duration) error {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if CheckFullDiskAccess() {
				return nil
			}
		}
	}
}

// GetCurrentAppPermissions returns permissions for the current application
func GetCurrentAppPermissions() ([]TCCEntry, error) {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Check if we're in a bundle
	bundlePath := system.GetContainingBundle(execPath)
	clientID := execPath
	if bundlePath != "" {
		// Use bundle identifier if we're in a bundle
		if bundleID := system.GetBundleID(bundlePath); bundleID != "" {
			clientID = bundleID
		}
	}

	db, err := OpenTCCDatabase()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	return db.ListPermissionsForClient(clientID)
}

// CheckPermission checks if the current app has a specific permission
func CheckPermission(service string) (bool, error) {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Check if we're in a bundle
	bundlePath := system.GetContainingBundle(execPath)
	clientID := execPath
	if bundlePath != "" {
		// Use bundle identifier if we're in a bundle
		if bundleID := system.GetBundleID(bundlePath); bundleID != "" {
			clientID = bundleID
		}
	}

	db, err := OpenTCCDatabase()
	if err != nil {
		// If we can't open the database, we don't have FDA
		if strings.Contains(err.Error(), "Full Disk Access") {
			return false, fmt.Errorf("cannot check permissions without Full Disk Access")
		}
		return false, err
	}
	defer db.Close()

	status, err := db.GetPermissionStatus(service, clientID)
	if err != nil {
		return false, err
	}

	return status.Granted, nil
}

// FormatTCCEntries formats TCC entries as a string table
func FormatTCCEntries(entries []TCCEntry, format string) (string, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil

	case "table", "":
		var b strings.Builder
		b.WriteString("Service                     | Client                      | Allowed | Last Used\n")
		b.WriteString("---------------------------|---------------------------|---------|--------------------\n")

		for _, entry := range entries {
			service := entry.Service
			if len(service) > 26 {
				service = service[:23] + "..."
			}

			client := entry.Client
			// Extract just the app name from bundle paths
			if strings.Contains(client, ".app") {
				parts := strings.Split(client, "/")
				for _, part := range parts {
					if strings.HasSuffix(part, ".app") {
						client = strings.TrimSuffix(part, ".app")
						break
					}
				}
			} else if strings.Contains(client, ".") && !strings.Contains(client, "/") {
				// Likely a bundle ID
				parts := strings.Split(client, ".")
				if len(parts) > 0 {
					client = parts[len(parts)-1]
				}
			}

			if len(client) > 26 {
				client = client[:23] + "..."
			}

			allowed := "No"
			if entry.Allowed {
				allowed = "Yes"
			}

			lastUsed := ""
			if !entry.LastUsed.IsZero() {
				lastUsed = entry.LastUsed.Format("2006-01-02 15:04:05")
			}

			b.WriteString(fmt.Sprintf("%-26s | %-26s | %-7s | %s\n",
				service, client, allowed, lastUsed))
		}

		return b.String(), nil

	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// TCCService represents a known TCC service
type TCCService struct {
	Name        string
	Description string
	Required    bool // Whether this permission is commonly required
}

// KnownTCCServices returns a list of known TCC services
func KnownTCCServices() []TCCService {
	return []TCCService{
		{"kTCCServiceCamera", "Camera access", true},
		{"kTCCServiceMicrophone", "Microphone access", true},
		{"kTCCServiceLocation", "Location services", false},
		{"kTCCServiceAddressBook", "Contacts access", false},
		{"kTCCServiceCalendar", "Calendar access", false},
		{"kTCCServiceReminders", "Reminders access", false},
		{"kTCCServicePhotos", "Photos library access", false},
		{"kTCCServiceScreenCapture", "Screen recording", false},
		{"kTCCServiceAccessibility", "Accessibility features", false},
		{"kTCCServicePostEvent", "Post events to other apps", false},
		{"kTCCServiceSystemPolicyAllFiles", "Full Disk Access", false},
		{"kTCCServiceSystemPolicyDesktopFolder", "Desktop folder access", false},
		{"kTCCServiceSystemPolicyDocumentsFolder", "Documents folder access", false},
		{"kTCCServiceSystemPolicyDownloadsFolder", "Downloads folder access", false},
		{"kTCCServiceSystemPolicyNetworkVolumes", "Network volumes access", false},
		{"kTCCServiceSystemPolicyRemovableVolumes", "Removable volumes access", false},
		{"kTCCServiceAppleEvents", "Apple Events (automation)", false},
		{"kTCCServiceBluetoothAlways", "Bluetooth access", false},
	}
}

// GetServiceDescription returns a human-readable description of a TCC service
func GetServiceDescription(service string) string {
	for _, s := range KnownTCCServices() {
		if s.Name == service {
			return s.Description
		}
	}
	// Clean up the service name if not found
	service = strings.TrimPrefix(service, "kTCCService")
	service = strings.TrimPrefix(service, "SystemPolicy")
	// Add spaces before capitals
	var result strings.Builder
	for i, r := range service {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteRune(r)
	}
	return result.String()
}