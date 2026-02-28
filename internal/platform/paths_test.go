package platform

import (
	"path/filepath"
	"testing"
)

// TestPathsForLinuxWithXDG verifies behavior for the covered scenario.
func TestPathsForLinuxWithXDG(t *testing.T) {
	p, err := PathsFor("linux", map[string]string{
		"XDG_CONFIG_HOME": "/xdg/config",
		"XDG_DATA_HOME":   "/xdg/data",
	}, "/fallback/config", "/fallback/data", "hakoll")
	if err != nil {
		t.Fatalf("PathsFor() error = %v", err)
	}
	wantConfig := filepath.Join("/xdg/config", "hakoll", "config.toml")
	wantDB := filepath.Join("/xdg/data", "hakoll", "hakoll.db")
	if p.ConfigPath != wantConfig {
		t.Fatalf("unexpected config path %q", p.ConfigPath)
	}
	if p.DBPath != wantDB {
		t.Fatalf("unexpected db path %q", p.DBPath)
	}
}

// TestPathsForWindowsUsesAppData verifies behavior for the covered scenario.
func TestPathsForWindowsUsesAppData(t *testing.T) {
	p, err := PathsFor("windows", map[string]string{
		"APPDATA":      `C:\Users\me\AppData\Roaming`,
		"LOCALAPPDATA": `C:\Users\me\AppData\Local`,
	}, `C:\fallback\config`, `C:\fallback\data`, "hakoll")
	if err != nil {
		t.Fatalf("PathsFor() error = %v", err)
	}

	wantConfig := filepath.Join(`C:\Users\me\AppData\Roaming`, "hakoll", "config.toml")
	wantDB := filepath.Join(`C:\Users\me\AppData\Local`, "hakoll", "hakoll.db")
	if p.ConfigPath != wantConfig {
		t.Fatalf("unexpected config path %q", p.ConfigPath)
	}
	if p.DBPath != wantDB {
		t.Fatalf("unexpected db path %q", p.DBPath)
	}
}

// TestPathsForEmptyDirsFails verifies behavior for the covered scenario.
func TestPathsForEmptyDirsFails(t *testing.T) {
	_, err := PathsFor("darwin", nil, "", "/tmp/data", "hakoll")
	if err == nil {
		t.Fatal("expected error for empty dirs")
	}
}

// TestPathsForDarwinFallback verifies behavior for the covered scenario.
func TestPathsForDarwinFallback(t *testing.T) {
	p, err := PathsFor("darwin", map[string]string{
		"XDG_CONFIG_HOME": "/ignored",
		"XDG_DATA_HOME":   "/ignored",
	}, "/Users/me/Library/Application Support", "/Users/me/Library/Application Support", "hakoll")
	if err != nil {
		t.Fatalf("PathsFor() error = %v", err)
	}
	wantConfig := filepath.Join("/Users/me/Library/Application Support", "hakoll", "config.toml")
	wantDB := filepath.Join("/Users/me/Library/Application Support", "hakoll", "hakoll.db")
	if p.ConfigPath != wantConfig {
		t.Fatalf("unexpected config path %q", p.ConfigPath)
	}
	if p.DBPath != wantDB {
		t.Fatalf("unexpected db path %q", p.DBPath)
	}
}

// TestPathsForUnknownFallback verifies behavior for the covered scenario.
func TestPathsForUnknownFallback(t *testing.T) {
	p, err := PathsFor("freebsd", map[string]string{}, "/cfg", "/data", "hakoll")
	if err != nil {
		t.Fatalf("PathsFor() error = %v", err)
	}
	wantConfig := filepath.Join("/cfg", "hakoll", "config.toml")
	wantData := filepath.Join("/data", "hakoll")
	if p.ConfigPath != wantConfig {
		t.Fatalf("unexpected config path %q", p.ConfigPath)
	}
	if p.DataDir != wantData {
		t.Fatalf("unexpected data dir %q", p.DataDir)
	}
}

// TestPathsForLinuxFallbackWithoutXDG verifies behavior for the covered scenario.
func TestPathsForLinuxFallbackWithoutXDG(t *testing.T) {
	p, err := PathsFor("linux", map[string]string{}, "/home/me/.config", "/home/me/.local/share", "hakoll")
	if err != nil {
		t.Fatalf("PathsFor() error = %v", err)
	}
	wantConfig := filepath.Join("/home/me/.config", "hakoll", "config.toml")
	wantDB := filepath.Join("/home/me/.local/share", "hakoll", "hakoll.db")
	if p.ConfigPath != wantConfig {
		t.Fatalf("unexpected config path %q", p.ConfigPath)
	}
	if p.DBPath != wantDB {
		t.Fatalf("unexpected db path %q", p.DBPath)
	}
}

// TestDefaultPathsSmoke verifies behavior for the covered scenario.
func TestDefaultPathsSmoke(t *testing.T) {
	p, err := DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths() error = %v", err)
	}
	if p.ConfigPath == "" || p.DBPath == "" || p.DataDir == "" {
		t.Fatalf("expected non-empty paths, got %#v", p)
	}
}

// TestDefaultPathsWithOptionsDevMode verifies behavior for the covered scenario.
func TestDefaultPathsWithOptionsDevMode(t *testing.T) {
	p, err := DefaultPathsWithOptions(Options{AppName: "hakoll", DevMode: true})
	if err != nil {
		t.Fatalf("DefaultPathsWithOptions() error = %v", err)
	}
	if filepath.Base(filepath.Dir(p.ConfigPath)) != "hakoll-dev" {
		t.Fatalf("expected dev config dir suffix, got %q", p.ConfigPath)
	}
	if filepath.Base(p.DBPath) != "hakoll-dev.db" {
		t.Fatalf("expected dev db name, got %q", p.DBPath)
	}
}
