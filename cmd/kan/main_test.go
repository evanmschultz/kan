package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/evanschultz/kan/internal/app"
	"github.com/evanschultz/kan/internal/config"
	"github.com/evanschultz/kan/internal/domain"
)

// TestMain sets deterministic environment defaults for CLI tests.
func TestMain(m *testing.M) {
	_ = os.Setenv("KAN_DEV_MODE", "false")
	os.Exit(m.Run())
}

// fakeProgram represents fake program data used by this package.
type fakeProgram struct {
	runErr error
}

// Run runs the requested command flow.
func (f fakeProgram) Run() (tea.Model, error) {
	return nil, f.runErr
}

// TestRunVersion verifies behavior for the covered scenario.
func TestRunVersion(t *testing.T) {
	var out strings.Builder
	err := run(context.Background(), []string{"--version"}, &out, io.Discard)
	if err != nil {
		t.Fatalf("run(version) error = %v", err)
	}
	if !strings.Contains(out.String(), "kan") {
		t.Fatalf("expected version output, got %q", out.String())
	}
}

// TestRunStartsProgram verifies behavior for the covered scenario.
func TestRunStartsProgram(t *testing.T) {
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })

	programFactory = func(_ tea.Model) program {
		return fakeProgram{}
	}

	dbPath := filepath.Join(t.TempDir(), "kan.db")
	err := run(context.Background(), []string{"--db", dbPath, "--config", filepath.Join(t.TempDir(), "missing.toml")}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
}

// TestRunTUIStartupDoesNotCreateDefaultProject verifies behavior for the covered scenario.
func TestRunTUIStartupDoesNotCreateDefaultProject(t *testing.T) {
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })
	programFactory = func(_ tea.Model) program { return fakeProgram{} }

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "kan.db")
	cfgPath := filepath.Join(tmp, "missing.toml")
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	outPath := filepath.Join(tmp, "snapshot.json")
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "export", "--out", outPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(export) error = %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var snap app.Snapshot
	if err := json.Unmarshal(content, &snap); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(snap.Projects) != 0 {
		t.Fatalf("expected no auto-created startup projects, got %d", len(snap.Projects))
	}
}

// TestRunInvalidFlag verifies behavior for the covered scenario.
func TestRunInvalidFlag(t *testing.T) {
	err := run(context.Background(), []string{"--unknown-flag"}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("expected flag parse error")
	}
}

// TestRunUnknownCommand verifies behavior for the covered scenario.
func TestRunUnknownCommand(t *testing.T) {
	err := run(context.Background(), []string{"unknown-command"}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got %v", err)
	}
}

// TestRunExportCommandWritesSnapshot verifies behavior for the covered scenario.
func TestRunExportCommandWritesSnapshot(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "kan.db")
	cfgPath := filepath.Join(tmp, "missing.toml")
	outPath := filepath.Join(tmp, "snapshot.json")
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "export", "--out", outPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(export) error = %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var snap app.Snapshot
	if err := json.Unmarshal(content, &snap); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if snap.Version != app.SnapshotVersion {
		t.Fatalf("unexpected snapshot version %q", snap.Version)
	}
	if len(snap.Projects) != 0 {
		t.Fatalf("expected no projects in empty export snapshot, got %d", len(snap.Projects))
	}
}

// TestRunImportCommandReadsSnapshot verifies behavior for the covered scenario.
func TestRunImportCommandReadsSnapshot(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "kan.db")
	cfgPath := filepath.Join(tmp, "missing.toml")

	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	snap := app.Snapshot{
		Version: app.SnapshotVersion,
		Projects: []app.SnapshotProject{
			{
				ID:        "p-import",
				Slug:      "imported",
				Name:      "Imported",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		Columns: []app.SnapshotColumn{
			{
				ID:        "c-import",
				ProjectID: "p-import",
				Name:      "To Do",
				Position:  0,
				WIPLimit:  0,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		Tasks: []app.SnapshotTask{
			{
				ID:        "t-import",
				ProjectID: "p-import",
				ColumnID:  "c-import",
				Position:  0,
				Title:     "Imported Task",
				Priority:  domain.PriorityMedium,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	content, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() error = %v", err)
	}
	inPath := filepath.Join(tmp, "in.json")
	if err := os.WriteFile(inPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "import", "--in", inPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(import) error = %v", err)
	}

	outPath := filepath.Join(tmp, "out.json")
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "export", "--out", outPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(export) error = %v", err)
	}
	outContent, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var outSnap app.Snapshot
	if err := json.Unmarshal(outContent, &outSnap); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	foundProject := false
	foundTask := false
	for _, p := range outSnap.Projects {
		if p.ID == "p-import" {
			foundProject = true
			break
		}
	}
	for _, tk := range outSnap.Tasks {
		if tk.ID == "t-import" {
			foundTask = true
			break
		}
	}
	if !foundProject || !foundTask {
		t.Fatalf("expected imported data in exported snapshot, foundProject=%t foundTask=%t", foundProject, foundTask)
	}
}

// TestRunExportToStdoutAndImportErrors verifies behavior for the covered scenario.
func TestRunExportToStdoutAndImportErrors(t *testing.T) {
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })
	programFactory = func(_ tea.Model) program { return fakeProgram{} }

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "kan.db")
	cfgPath := filepath.Join(tmp, "missing.toml")
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("initial run() error = %v", err)
	}

	var out strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "export", "--out", "-"}, &out, io.Discard); err != nil {
		t.Fatalf("run(export stdout) error = %v", err)
	}
	if !strings.Contains(out.String(), "\"version\"") {
		t.Fatalf("expected snapshot json on stdout, got %q", out.String())
	}

	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "import"}, io.Discard, io.Discard); err == nil {
		t.Fatal("expected import error for missing --in")
	}

	badIn := filepath.Join(tmp, "bad.json")
	if err := os.WriteFile(badIn, []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "import", "--in", badIn}, io.Discard, io.Discard); err == nil {
		t.Fatal("expected import decode error")
	}
}

// TestRunConfigAndDBEnvOverrides verifies behavior for the covered scenario.
func TestRunConfigAndDBEnvOverrides(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "env.db")
	cfgPath := filepath.Join(tmp, "env.toml")
	cfgContent := "[database]\npath = \"/tmp/ignore-me.db\"\n"
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("KAN_CONFIG", cfgPath)
	t.Setenv("KAN_DB_PATH", dbPath)

	err := run(context.Background(), []string{"export", "--out", filepath.Join(tmp, "out.json")}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("run(export with env paths) error = %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected db created at env path, stat error %v", err)
	}
}

// TestRunPathsCommand verifies behavior for the covered scenario.
func TestRunPathsCommand(t *testing.T) {
	var out strings.Builder
	err := run(context.Background(), []string{"--app", "kanx", "--dev", "paths"}, &out, io.Discard)
	if err != nil {
		t.Fatalf("run(paths) error = %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "app: kanx") {
		t.Fatalf("expected app name in paths output, got %q", output)
	}
	if !strings.Contains(output, "dev_mode: true") {
		t.Fatalf("expected dev mode in paths output, got %q", output)
	}
}

// TestParseBoolEnv verifies behavior for the covered scenario.
func TestParseBoolEnv(t *testing.T) {
	t.Setenv("KAN_BOOL_TEST", "true")
	got, ok := parseBoolEnv("KAN_BOOL_TEST")
	if !ok || !got {
		t.Fatalf("expected true bool env parse, got value=%t ok=%t", got, ok)
	}

	t.Setenv("KAN_BOOL_TEST", "not-bool")
	_, ok = parseBoolEnv("KAN_BOOL_TEST")
	if ok {
		t.Fatal("expected invalid bool env to return ok=false")
	}
}

// TestRunDevModeCreatesWorkspaceLogFile verifies behavior for the covered scenario.
func TestRunDevModeCreatesWorkspaceLogFile(t *testing.T) {
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })
	programFactory = func(_ tea.Model) program { return fakeProgram{} }

	workspace := t.TempDir()
	t.Chdir(workspace)

	dbPath := filepath.Join(workspace, "kan.db")
	cfgPath := filepath.Join(workspace, "missing.toml")
	if err := run(context.Background(), []string{"--dev", "--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	logDir := filepath.Join(workspace, ".kan", "log")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected dev log file in %s", logDir)
	}
	foundLog := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
			foundLog = true
			break
		}
	}
	if !foundLog {
		t.Fatalf("expected at least one .log file in %s, got %v", logDir, entries)
	}
}

// TestWorkspaceRootFromUsesNearestMarker verifies workspace-root resolution behavior.
func TestWorkspaceRootFromUsesNearestMarker(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	nested := filepath.Join(root, "cmd", "kan")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	got := workspaceRootFrom(nested)
	if filepath.Clean(got) != filepath.Clean(root) {
		t.Fatalf("expected workspace root %q, got %q", root, got)
	}
}

// TestDevLogFilePathResolvesAgainstWorkspaceRoot verifies relative log dirs anchor at workspace root.
func TestDevLogFilePathResolvesAgainstWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	nested := filepath.Join(root, "cmd", "kan")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prev)
	})
	got, err := devLogFilePath(".kan/log", "kan", time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("devLogFilePath() error = %v", err)
	}
	wantPrefix := filepath.Join(root, ".kan", "log")
	normalize := func(p string) string {
		return strings.TrimPrefix(filepath.Clean(p), "/private")
	}
	if !strings.HasPrefix(normalize(got), normalize(wantPrefix)) {
		t.Fatalf("expected log path under %q, got %q", wantPrefix, got)
	}
}

// TestRunRejectsInvalidLoggingLevelFromConfig verifies behavior for the covered scenario.
func TestRunRejectsInvalidLoggingLevelFromConfig(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "kan.db")
	cfgPath := filepath.Join(tmp, "kan.toml")
	cfgContent := "[logging]\nlevel = \"verbose\"\n"
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("expected invalid logging level error")
	}
	if !strings.Contains(err.Error(), "invalid logging.level") {
		t.Fatalf("expected logging level validation error, got %v", err)
	}
}

// TestLoadRuntimeConfigMapsRuntimeFields verifies behavior for the covered scenario.
func TestLoadRuntimeConfigMapsRuntimeFields(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "kan.toml")
	content := `
[database]
path = "/tmp/from-config.db"

[delete]
default_mode = "hard"

[confirm]
delete = false
archive = false
hard_delete = false
restore = true

[task_fields]
show_priority = false
show_due_date = false
show_labels = false
show_description = true

[board]
show_wip_warnings = false
group_by = "priority"

[search]
cross_project = true
include_archived = true
states = ["todo", "archived"]

[ui]
due_soon_windows = ["6h"]
show_due_summary = false

[project_roots]
inbox = "/tmp/inbox"

[labels]
global = ["bug"]
enforce_allowed = true

[labels.projects]
inbox = ["roadmap"]

[keys]
command_palette = ";"
quick_actions = ","
multi_select = "x"
activity_log = "v"
undo = "u"
redo = "U"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	runtimeCfg, err := loadRuntimeConfig(cfgPath, config.Default("/tmp/default.db"), "/tmp/override.db", true)
	if err != nil {
		t.Fatalf("loadRuntimeConfig() error = %v", err)
	}
	if runtimeCfg.DefaultDeleteMode != app.DeleteModeHard {
		t.Fatalf("expected hard delete mode, got %q", runtimeCfg.DefaultDeleteMode)
	}
	if runtimeCfg.TaskFields.ShowPriority || runtimeCfg.TaskFields.ShowDueDate || runtimeCfg.TaskFields.ShowLabels || !runtimeCfg.TaskFields.ShowDescription {
		t.Fatalf("unexpected task fields runtime config %#v", runtimeCfg.TaskFields)
	}
	if !runtimeCfg.Search.CrossProject || !runtimeCfg.Search.IncludeArchived {
		t.Fatalf("unexpected search runtime config %#v", runtimeCfg.Search)
	}
	if runtimeCfg.Board.GroupBy != "priority" || runtimeCfg.Board.ShowWIPWarnings {
		t.Fatalf("unexpected board runtime config %#v", runtimeCfg.Board)
	}
	if runtimeCfg.Confirm.Delete || runtimeCfg.Confirm.Archive || runtimeCfg.Confirm.HardDelete || !runtimeCfg.Confirm.Restore {
		t.Fatalf("unexpected confirm runtime config %#v", runtimeCfg.Confirm)
	}
	if len(runtimeCfg.UI.DueSoonWindows) != 1 || runtimeCfg.UI.DueSoonWindows[0] != 6*time.Hour || runtimeCfg.UI.ShowDueSummary {
		t.Fatalf("unexpected ui runtime config %#v", runtimeCfg.UI)
	}
	if got := runtimeCfg.Keys.CommandPalette; got != ";" {
		t.Fatalf("expected command palette key override ';', got %q", got)
	}
	if got := runtimeCfg.ProjectRoots["inbox"]; got != "/tmp/inbox" {
		t.Fatalf("unexpected project roots runtime config %#v", runtimeCfg.ProjectRoots)
	}
	if !runtimeCfg.Labels.EnforceAllowed || len(runtimeCfg.Labels.Global) != 1 || runtimeCfg.Labels.Global[0] != "bug" {
		t.Fatalf("unexpected label runtime config %#v", runtimeCfg.Labels)
	}
	if got := runtimeCfg.Labels.Projects["inbox"]; len(got) != 1 || got[0] != "roadmap" {
		t.Fatalf("unexpected project labels runtime config %#v", runtimeCfg.Labels.Projects)
	}
}

// TestPersistProjectRootRoundTrip verifies behavior for the covered scenario.
func TestPersistProjectRootRoundTrip(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "kan.toml")

	if err := persistProjectRoot(cfgPath, "Inbox", "/tmp/inbox"); err != nil {
		t.Fatalf("persistProjectRoot() error = %v", err)
	}
	cfg, err := config.Load(cfgPath, config.Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.ProjectRoots["inbox"]; got != "/tmp/inbox" {
		t.Fatalf("expected persisted project root /tmp/inbox, got %#v", cfg.ProjectRoots)
	}

	if err := persistProjectRoot(cfgPath, "inbox", ""); err != nil {
		t.Fatalf("persistProjectRoot(clear) error = %v", err)
	}
	cfg, err = config.Load(cfgPath, config.Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() after clear error = %v", err)
	}
	if _, ok := cfg.ProjectRoots["inbox"]; ok {
		t.Fatalf("expected project root cleared, got %#v", cfg.ProjectRoots)
	}
}

// TestPersistAllowedLabelsRoundTrip verifies behavior for the covered scenario.
func TestPersistAllowedLabelsRoundTrip(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "kan.toml")

	if err := persistAllowedLabels(cfgPath, "Inbox", []string{"Bug", "chore", "bug"}, []string{"Roadmap", "kan", "roadmap"}); err != nil {
		t.Fatalf("persistAllowedLabels() error = %v", err)
	}
	cfg, err := config.Load(cfgPath, config.Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	wantGlobal := []string{"bug", "chore"}
	if len(cfg.Labels.Global) != len(wantGlobal) {
		t.Fatalf("unexpected persisted global labels %#v", cfg.Labels.Global)
	}
	for i := range wantGlobal {
		if cfg.Labels.Global[i] != wantGlobal[i] {
			t.Fatalf("unexpected global label at %d: got %q want %q", i, cfg.Labels.Global[i], wantGlobal[i])
		}
	}
	wantProject := []string{"kan", "roadmap"}
	gotProject := cfg.Labels.Projects["inbox"]
	if len(gotProject) != len(wantProject) {
		t.Fatalf("unexpected persisted project labels %#v", cfg.Labels.Projects)
	}
	for i := range wantProject {
		if gotProject[i] != wantProject[i] {
			t.Fatalf("unexpected project label at %d: got %q want %q", i, gotProject[i], wantProject[i])
		}
	}

	if err := persistAllowedLabels(cfgPath, "inbox", []string{"bug"}, nil); err != nil {
		t.Fatalf("persistAllowedLabels(clear project labels) error = %v", err)
	}
	cfg, err = config.Load(cfgPath, config.Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() after clear error = %v", err)
	}
	if _, ok := cfg.Labels.Projects["inbox"]; ok {
		t.Fatalf("expected inbox project labels cleared, got %#v", cfg.Labels.Projects)
	}
	if len(cfg.Labels.Global) != 1 || cfg.Labels.Global[0] != "bug" {
		t.Fatalf("expected global labels to remain bug, got %#v", cfg.Labels.Global)
	}
}
