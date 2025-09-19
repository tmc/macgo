package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a test source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	srcContent := []byte("test content for file copy")
	if err := os.WriteFile(srcPath, srcContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		src     string
		dst     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "successful copy",
			src:     srcPath,
			dst:     filepath.Join(tmpDir, "dest.txt"),
			wantErr: false,
		},
		{
			name:    "copy to subdirectory",
			src:     srcPath,
			dst:     filepath.Join(tmpDir, "subdir", "dest.txt"),
			wantErr: false,
		},
		{
			name:    "empty source path",
			src:     "",
			dst:     filepath.Join(tmpDir, "dest.txt"),
			wantErr: true,
			errMsg:  "source file path cannot be empty",
		},
		{
			name:    "empty destination path",
			src:     srcPath,
			dst:     "",
			wantErr: true,
			errMsg:  "destination file path cannot be empty",
		},
		{
			name:    "non-existent source",
			src:     filepath.Join(tmpDir, "nonexistent.txt"),
			dst:     filepath.Join(tmpDir, "dest.txt"),
			wantErr: true,
			errMsg:  "failed to open source file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CopyFile(tt.src, tt.dst)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CopyFile() expected error but got none")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("CopyFile() error = %v, want to contain %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("CopyFile() error = %v, want nil", err)
					return
				}

				// Verify the file was copied correctly
				dstContent, err := os.ReadFile(tt.dst)
				if err != nil {
					t.Errorf("Failed to read destination file: %v", err)
					return
				}

				if string(dstContent) != string(srcContent) {
					t.Errorf("File content mismatch: got %q, want %q", string(dstContent), string(srcContent))
				}

				// Verify permissions were preserved
				srcInfo, err := os.Stat(tt.src)
				if err != nil {
					t.Errorf("Failed to get source file info: %v", err)
					return
				}

				dstInfo, err := os.Stat(tt.dst)
				if err != nil {
					t.Errorf("Failed to get destination file info: %v", err)
					return
				}

				if srcInfo.Mode() != dstInfo.Mode() {
					t.Errorf("File permissions not preserved: got %v, want %v", dstInfo.Mode(), srcInfo.Mode())
				}
			}
		})
	}
}

func TestIsInAppBundle(t *testing.T) {
	// We can't easily test this without actually being in an app bundle,
	// but we can test that it doesn't panic and returns a boolean
	result := IsInAppBundle()
	t.Logf("IsInAppBundle() = %v", result)
	// In normal test execution, this should be false
	if result {
		t.Log("Note: Tests appear to be running inside an app bundle")
	}
}

func TestSafeWriteFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		data     []byte
		perm     os.FileMode
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "successful write",
			filename: filepath.Join(tmpDir, "test.txt"),
			data:     []byte("test content"),
			perm:     0644,
			wantErr:  false,
		},
		{
			name:     "write to subdirectory",
			filename: filepath.Join(tmpDir, "subdir", "test.txt"),
			data:     []byte("test content"),
			perm:     0644,
			wantErr:  false,
		},
		{
			name:     "empty filename",
			filename: "",
			data:     []byte("test content"),
			perm:     0644,
			wantErr:  true,
			errMsg:   "filename cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SafeWriteFile(tt.filename, tt.data, tt.perm)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SafeWriteFile() expected error but got none")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("SafeWriteFile() error = %v, want to contain %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SafeWriteFile() error = %v, want nil", err)
					return
				}

				// Verify the file was written correctly
				content, err := os.ReadFile(tt.filename)
				if err != nil {
					t.Errorf("Failed to read written file: %v", err)
					return
				}

				if string(content) != string(tt.data) {
					t.Errorf("File content mismatch: got %q, want %q", string(content), string(tt.data))
				}

				// Verify permissions
				info, err := os.Stat(tt.filename)
				if err != nil {
					t.Errorf("Failed to get file info: %v", err)
					return
				}

				if info.Mode() != tt.perm {
					t.Errorf("File permissions mismatch: got %v, want %v", info.Mode(), tt.perm)
				}
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a test directory
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "existing file",
			path: testFile,
			want: true,
		},
		{
			name: "existing directory",
			path: testDir,
			want: false, // FileExists should return false for directories
		},
		{
			name: "non-existent path",
			path: filepath.Join(tmpDir, "nonexistent.txt"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FileExists(tt.path)
			if got != tt.want {
				t.Errorf("FileExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a test directory
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "existing directory",
			path: testDir,
			want: true,
		},
		{
			name: "existing file",
			path: testFile,
			want: false, // DirExists should return false for files
		},
		{
			name: "non-existent path",
			path: filepath.Join(tmpDir, "nonexistent"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DirExists(tt.path)
			if got != tt.want {
				t.Errorf("DirExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		perm    os.FileMode
		wantErr bool
		errMsg  string
	}{
		{
			name:    "create single directory",
			path:    filepath.Join(tmpDir, "newdir"),
			perm:    0755,
			wantErr: false,
		},
		{
			name:    "create nested directories",
			path:    filepath.Join(tmpDir, "nested", "path", "dir"),
			perm:    0755,
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			perm:    0755,
			wantErr: true,
			errMsg:  "path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureDir(tt.path, tt.perm)

			if tt.wantErr {
				if err == nil {
					t.Errorf("EnsureDir() expected error but got none")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("EnsureDir() error = %v, want to contain %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("EnsureDir() error = %v, want nil", err)
					return
				}

				// Verify the directory was created
				if !DirExists(tt.path) {
					t.Errorf("Directory was not created: %s", tt.path)
				}
			}
		})
	}
}

func TestBundlePaths(t *testing.T) {
	bundlePath := "/path/to/MyApp.app"
	execName := "MyApp"

	tests := []struct {
		name string
		fn   func() string
		want string
	}{
		{
			name: "GetBundleExecutablePath",
			fn:   func() string { return GetBundleExecutablePath(bundlePath, execName) },
			want: "/path/to/MyApp.app/Contents/MacOS/MyApp",
		},
		{
			name: "GetBundleContentsPath",
			fn:   func() string { return GetBundleContentsPath(bundlePath) },
			want: "/path/to/MyApp.app/Contents",
		},
		{
			name: "GetBundleInfoPlistPath",
			fn:   func() string { return GetBundleInfoPlistPath(bundlePath) },
			want: "/path/to/MyApp.app/Contents/Info.plist",
		},
		{
			name: "GetBundleEntitlementsPath",
			fn:   func() string { return GetBundleEntitlementsPath(bundlePath) },
			want: "/path/to/MyApp.app/Contents/entitlements.plist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsAppBundle(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a proper app bundle structure
	appBundlePath := filepath.Join(tmpDir, "Test.app")
	contentsPath := filepath.Join(appBundlePath, "Contents")
	if err := os.MkdirAll(contentsPath, 0755); err != nil {
		t.Fatalf("Failed to create app bundle structure: %v", err)
	}

	// Create a fake app bundle (just .app directory without Contents)
	fakeAppPath := filepath.Join(tmpDir, "Fake.app")
	if err := os.Mkdir(fakeAppPath, 0755); err != nil {
		t.Fatalf("Failed to create fake app bundle: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "valid app bundle",
			path: appBundlePath,
			want: true,
		},
		{
			name: "fake app bundle",
			path: fakeAppPath,
			want: false,
		},
		{
			name: "regular directory",
			path: tmpDir,
			want: false,
		},
		{
			name: "non-existent path",
			path: filepath.Join(tmpDir, "nonexistent.app"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAppBundle(tt.path)
			if got != tt.want {
				t.Errorf("IsAppBundle() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions for string operations
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}()))
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
