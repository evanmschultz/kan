package platform

import (
	"path/filepath"
	"testing"
)

func TestPathsForLinuxWithXDG(t *testing.T) {
	p, err := PathsFor("linux", map[string]string{
		"XDG_CONFIG_HOME": "/xdg/config",
		"XDG_DATA_HOME":   "/xdg/data",
	}, "/fallback/config", "/fallback/data")
	if err != nil {
		t.Fatalf("PathsFor() error = %v", err)
	}
	if p.ConfigPath != "/xdg/config/kan/config.toml" {
		t.Fatalf("unexpected config path %q", p.ConfigPath)
	}
	if p.DBPath != "/xdg/data/kan/kan.db" {
		t.Fatalf("unexpected db path %q", p.DBPath)
	}
}

func TestPathsForWindowsUsesAppData(t *testing.T) {
	p, err := PathsFor("windows", map[string]string{
		"APPDATA":      `C:\Users\me\AppData\Roaming`,
		"LOCALAPPDATA": `C:\Users\me\AppData\Local`,
	}, `C:\fallback\config`, `C:\fallback\data`)
	if err != nil {
		t.Fatalf("PathsFor() error = %v", err)
	}

	wantConfig := filepath.Join(`C:\Users\me\AppData\Roaming`, "kan", "config.toml")
	wantDB := filepath.Join(`C:\Users\me\AppData\Local`, "kan", "kan.db")
	if p.ConfigPath != wantConfig {
		t.Fatalf("unexpected config path %q", p.ConfigPath)
	}
	if p.DBPath != wantDB {
		t.Fatalf("unexpected db path %q", p.DBPath)
	}
}

func TestPathsForEmptyDirsFails(t *testing.T) {
	_, err := PathsFor("darwin", nil, "", "/tmp/data")
	if err == nil {
		t.Fatal("expected error for empty dirs")
	}
}

func TestPathsForDarwinFallback(t *testing.T) {
	p, err := PathsFor("darwin", map[string]string{
		"XDG_CONFIG_HOME": "/ignored",
		"XDG_DATA_HOME":   "/ignored",
	}, "/Users/me/Library/Application Support", "/Users/me/Library/Caches")
	if err != nil {
		t.Fatalf("PathsFor() error = %v", err)
	}
	if p.ConfigPath != "/Users/me/Library/Application Support/kan/config.toml" {
		t.Fatalf("unexpected config path %q", p.ConfigPath)
	}
	if p.DBPath != "/Users/me/Library/Caches/kan/kan.db" {
		t.Fatalf("unexpected db path %q", p.DBPath)
	}
}

func TestPathsForUnknownFallback(t *testing.T) {
	p, err := PathsFor("freebsd", map[string]string{}, "/cfg", "/data")
	if err != nil {
		t.Fatalf("PathsFor() error = %v", err)
	}
	if p.ConfigPath != "/cfg/kan/config.toml" {
		t.Fatalf("unexpected config path %q", p.ConfigPath)
	}
	if p.DataDir != "/data/kan" {
		t.Fatalf("unexpected data dir %q", p.DataDir)
	}
}

func TestPathsForLinuxFallbackWithoutXDG(t *testing.T) {
	p, err := PathsFor("linux", map[string]string{}, "/home/me/.config", "/home/me/.local/share")
	if err != nil {
		t.Fatalf("PathsFor() error = %v", err)
	}
	if p.ConfigPath != "/home/me/.config/kan/config.toml" {
		t.Fatalf("unexpected config path %q", p.ConfigPath)
	}
	if p.DBPath != "/home/me/.local/share/kan/kan.db" {
		t.Fatalf("unexpected db path %q", p.DBPath)
	}
}

func TestDefaultPathsSmoke(t *testing.T) {
	p, err := DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths() error = %v", err)
	}
	if p.ConfigPath == "" || p.DBPath == "" || p.DataDir == "" {
		t.Fatalf("expected non-empty paths, got %#v", p)
	}
}
