package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDefaultConfig verifies behavior for the covered scenario.
func TestDefaultConfig(t *testing.T) {
	cfg := Default("/tmp/kan.db")
	if cfg.Database.Path != "/tmp/kan.db" {
		t.Fatalf("unexpected db path %q", cfg.Database.Path)
	}
	if cfg.Delete.DefaultMode != DeleteModeArchive {
		t.Fatalf("unexpected delete mode %q", cfg.Delete.DefaultMode)
	}
	if !cfg.Confirm.Delete || !cfg.Confirm.Archive || !cfg.Confirm.HardDelete {
		t.Fatalf("unexpected confirm defaults %#v", cfg.Confirm)
	}
	if cfg.Confirm.Restore {
		t.Fatalf("expected restore confirm disabled by default, got %#v", cfg.Confirm)
	}
	if !cfg.TaskFields.ShowPriority || !cfg.TaskFields.ShowDueDate || !cfg.TaskFields.ShowLabels {
		t.Fatal("expected priority/due_date/labels enabled by default")
	}
	if cfg.TaskFields.ShowDescription {
		t.Fatal("expected description disabled by default")
	}
	if got := cfg.UI.DueSoonWindows; len(got) != 2 || got[0] != "24h" || got[1] != "1h" {
		t.Fatalf("unexpected due windows %#v", got)
	}
	if !cfg.UI.ShowDueSummary {
		t.Fatal("expected due summary enabled by default")
	}
}

// TestLoadMissingFileUsesDefaults verifies behavior for the covered scenario.
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

// TestLoadFileOverridesDefaults verifies behavior for the covered scenario.
func TestLoadFileOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[database]
path = "/custom/kan.db"

[delete]
default_mode = "hard"

[confirm]
delete = true
archive = false
hard_delete = true
restore = true

[task_fields]
show_priority = true
show_due_date = false
show_labels = true
show_description = true

[ui]
due_soon_windows = ["12h", "45m"]
show_due_summary = false
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
	if cfg.Confirm.Archive {
		t.Fatalf("expected archive confirm false, got %#v", cfg.Confirm)
	}
	if cfg.UI.ShowDueSummary {
		t.Fatal("expected due summary hidden from config override")
	}
}

// TestLoadRejectsInvalidDeleteMode verifies behavior for the covered scenario.
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

// TestEnsureConfigDir verifies behavior for the covered scenario.
func TestEnsureConfigDir(t *testing.T) {
	target := filepath.Join(t.TempDir(), "a", "b", "config.toml")
	if err := EnsureConfigDir(target); err != nil {
		t.Fatalf("EnsureConfigDir() error = %v", err)
	}
	if _, err := os.Stat(filepath.Dir(target)); err != nil {
		t.Fatalf("expected dir to exist, stat error %v", err)
	}
}

// TestLoadBoardSearchAndKeysOverrides verifies behavior for the covered scenario.
func TestLoadBoardSearchAndKeysOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[database]
path = "/custom/kan.db"

[board]
show_wip_warnings = false
group_by = "priority"

[search]
cross_project = true
include_archived = true
states = ["todo", "progress", "archived"]

[ui]
due_soon_windows = ["2h", "48h"]
show_due_summary = true

[keys]
command_palette = ":"
quick_actions = "."
multi_select = "space"
activity_log = "g"
undo = "u"
redo = "U"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Board.GroupBy != "priority" || cfg.Board.ShowWIPWarnings {
		t.Fatalf("unexpected board settings %#v", cfg.Board)
	}
	if !cfg.Search.CrossProject || !cfg.Search.IncludeArchived {
		t.Fatalf("unexpected search settings %#v", cfg.Search)
	}
	if len(cfg.Search.States) != 3 {
		t.Fatalf("unexpected search states %#v", cfg.Search.States)
	}
	if cfg.Keys.QuickActions != "." {
		t.Fatalf("unexpected keys config %#v", cfg.Keys)
	}
	if got := cfg.DueSoonDurations(); len(got) != 2 || got[0] != 2*time.Hour || got[1] != 48*time.Hour {
		t.Fatalf("unexpected due durations %#v", got)
	}
}

// TestValidateRejectsUnknownSearchState verifies behavior for the covered scenario.
func TestValidateRejectsUnknownSearchState(t *testing.T) {
	cfg := Default("/tmp/kan.db")
	cfg.Search.States = []string{"todo", "unknown"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unknown search state validation error")
	}
}

// TestValidateRejectsInvalidDueSoonWindow verifies behavior for the covered scenario.
func TestValidateRejectsInvalidDueSoonWindow(t *testing.T) {
	cfg := Default("/tmp/kan.db")
	cfg.UI.DueSoonWindows = []string{"bogus"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid due-soon duration error")
	}
}

// TestDueSoonDurationsNormalizes verifies behavior for the covered scenario.
func TestDueSoonDurationsNormalizes(t *testing.T) {
	cfg := Default("/tmp/kan.db")
	cfg.UI.DueSoonWindows = []string{"2h", "30m", "2h", "bad", "0s"}
	got := cfg.DueSoonDurations()
	want := []time.Duration{30 * time.Minute, 2 * time.Hour}
	if len(got) != len(want) {
		t.Fatalf("unexpected due durations length %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected due duration at %d: got %s want %s", i, got[i], want[i])
		}
	}
}

// TestLoadProjectRootsAndLabels verifies behavior for the covered scenario.
func TestLoadProjectRootsAndLabels(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[database]
path = "/custom/kan.db"

[project_roots]
Inbox = "/Users/test/code/inbox"

[labels]
global = ["Planning", "Bug", "planning"]
enforce_allowed = true

[labels.projects]
inbox = ["kan", "Roadmap", "kan"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.ProjectRoots["inbox"]; got != "/Users/test/code/inbox" {
		t.Fatalf("unexpected project root mapping %#v", cfg.ProjectRoots)
	}
	if !cfg.Labels.EnforceAllowed {
		t.Fatalf("expected enforce_allowed true, got %#v", cfg.Labels)
	}
	allowed := cfg.AllowedLabels("inbox")
	want := []string{"bug", "kan", "planning", "roadmap"}
	if len(allowed) != len(want) {
		t.Fatalf("unexpected allowed labels %#v", allowed)
	}
	for i := range want {
		if allowed[i] != want[i] {
			t.Fatalf("unexpected allowed label at %d: got %q want %q", i, allowed[i], want[i])
		}
	}
}

// TestValidateRejectsEmptyProjectRoot verifies behavior for the covered scenario.
func TestValidateRejectsEmptyProjectRoot(t *testing.T) {
	cfg := Default("/tmp/kan.db")
	cfg.ProjectRoots["inbox"] = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty project root")
	}
}
