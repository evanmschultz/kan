package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default("/tmp/kan.db")
	if cfg.Database.Path != "/tmp/kan.db" {
		t.Fatalf("unexpected db path %q", cfg.Database.Path)
	}
	if cfg.Delete.DefaultMode != DeleteModeArchive {
		t.Fatalf("unexpected delete mode %q", cfg.Delete.DefaultMode)
	}
	if !cfg.TaskFields.ShowPriority || !cfg.TaskFields.ShowDueDate || !cfg.TaskFields.ShowLabels {
		t.Fatal("expected priority/due_date/labels enabled by default")
	}
	if cfg.TaskFields.ShowDescription {
		t.Fatal("expected description disabled by default")
	}
}

func TestLoadMissingFileUsesDefaults(t *testing.T) {
	defaults := Default("/tmp/kan.db")
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.toml"), defaults)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.Path != defaults.Database.Path {
		t.Fatalf("expected default db path, got %q", cfg.Database.Path)
	}
}

func TestLoadFileOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[database]
path = "/custom/kan.db"

[delete]
default_mode = "hard"

[task_fields]
show_priority = true
show_due_date = false
show_labels = true
show_description = true
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.Path != "/custom/kan.db" {
		t.Fatalf("unexpected db path %q", cfg.Database.Path)
	}
	if cfg.Delete.DefaultMode != DeleteModeHard {
		t.Fatalf("unexpected delete mode %q", cfg.Delete.DefaultMode)
	}
	if cfg.TaskFields.ShowDueDate {
		t.Fatal("expected due_date hidden from config override")
	}
	if !cfg.TaskFields.ShowDescription {
		t.Fatal("expected description visible from config override")
	}
}

func TestLoadRejectsInvalidDeleteMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[database]
path = "/custom/kan.db"

[delete]
default_mode = "weird"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err := Load(path, Default("/tmp/default.db"))
	if err == nil {
		t.Fatal("expected error for invalid delete mode")
	}
}

func TestEnsureConfigDir(t *testing.T) {
	target := filepath.Join(t.TempDir(), "a", "b", "config.toml")
	if err := EnsureConfigDir(target); err != nil {
		t.Fatalf("EnsureConfigDir() error = %v", err)
	}
	if _, err := os.Stat(filepath.Dir(target)); err != nil {
		t.Fatalf("expected dir to exist, stat error %v", err)
	}
}
