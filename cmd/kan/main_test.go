package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

// scriptedProgram represents program data used to exercise model flows inside run() tests.
type scriptedProgram struct {
	model tea.Model
	runFn func(tea.Model) (tea.Model, error)
}

// Run runs scripted model interactions and returns the final state.
func (p scriptedProgram) Run() (tea.Model, error) {
	if p.runFn == nil {
		return p.model, nil
	}
	return p.runFn(p.model)
}

// applyModelMsg applies one message and any resulting command chain.
func applyModelMsg(t *testing.T, model tea.Model, msg tea.Msg) tea.Model {
	t.Helper()
	updated, cmd := model.Update(msg)
	return applyModelCmd(t, updated, cmd)
}

// applyModelCmd executes one command chain to completion (bounded for safety).
func applyModelCmd(t *testing.T, model tea.Model, cmd tea.Cmd) tea.Model {
	t.Helper()
	out := model
	currentCmd := cmd
	for i := 0; i < 8 && currentCmd != nil; i++ {
		msg := currentCmd()
		updated, nextCmd := out.Update(msg)
		out = updated
		currentCmd = nextCmd
	}
	return out
}

// writeBootstrapReadyConfig writes the minimum startup fields required to bypass bootstrap modal gating.
func writeBootstrapReadyConfig(t *testing.T, path, searchRoot string) {
	t.Helper()
	content := fmt.Sprintf(`
[identity]
display_name = "Test User"
default_actor_type = "user"

[paths]
search_roots = [%q]
`, searchRoot)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
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
	cfgPath := filepath.Join(t.TempDir(), "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, t.TempDir())
	err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard)
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
	cfgPath := filepath.Join(tmp, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, t.TempDir())
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

// TestRunBootstrapModalPersistsMissingFields verifies startup bootstrap persists through TUI callbacks.
func TestRunBootstrapModalPersistsMissingFields(t *testing.T) {
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })
	programFactory = func(model tea.Model) program {
		return scriptedProgram{
			model: model,
			runFn: func(current tea.Model) (tea.Model, error) {
				current = applyModelCmd(t, current, current.Init())
				current = applyModelMsg(t, current, tea.WindowSizeMsg{Width: 120, Height: 40})
				if rendered := fmt.Sprint(current.View().Content); !strings.Contains(rendered, "Startup Setup Required") {
					t.Fatalf("expected startup bootstrap modal, got\n%s", rendered)
				}

				for _, r := range "Lane User" {
					current = applyModelMsg(t, current, tea.KeyPressMsg{Code: r, Text: string(r)})
				}
				current = applyModelMsg(t, current, tea.KeyPressMsg{Code: tea.KeyTab})
				current = applyModelMsg(t, current, tea.KeyPressMsg{Code: 'l', Text: "l"})
				current = applyModelMsg(t, current, tea.KeyPressMsg{Code: tea.KeyTab})
				current = applyModelMsg(t, current, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
				current = applyModelMsg(t, current, tea.KeyPressMsg{Code: 'a', Text: "a"})
				current = applyModelMsg(t, current, tea.KeyPressMsg{Code: tea.KeyTab})
				current = applyModelMsg(t, current, tea.KeyPressMsg{Code: tea.KeyEnter})
				if rendered := fmt.Sprint(current.View().Content); !strings.Contains(rendered, "Projects") {
					t.Fatalf("expected project picker after bootstrap save, got\n%s", rendered)
				}
				return current, nil
			},
		}
	}

	workspace := t.TempDir()
	t.Chdir(workspace)
	dbPath := filepath.Join(workspace, "kan.db")
	cfgPath := filepath.Join(workspace, "config.toml")

	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	cfg, err := config.Load(cfgPath, config.Default(dbPath))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.Identity.DisplayName; got != "Lane User" {
		t.Fatalf("expected persisted display name Lane User, got %q", got)
	}
	if got := cfg.Identity.DefaultActorType; got != "agent" {
		t.Fatalf("expected persisted actor type agent, got %q", got)
	}
	if len(cfg.Paths.SearchRoots) != 1 || cfg.Paths.SearchRoots[0] != filepath.Clean(workspace) {
		t.Fatalf("expected persisted search root %q, got %#v", filepath.Clean(workspace), cfg.Paths.SearchRoots)
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
	cfgPath := filepath.Join(tmp, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, t.TempDir())
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

// TestStartupBootstrapRequired verifies startup bootstrap requirement detection from config values.
func TestStartupBootstrapRequired(t *testing.T) {
	cfg := config.Default("/tmp/kan.db")
	cfg.Identity.DisplayName = ""
	cfg.Paths.SearchRoots = []string{"/tmp/code"}
	if !startupBootstrapRequired(cfg) {
		t.Fatal("expected missing display name to require startup bootstrap")
	}

	cfg.Identity.DisplayName = "Lane User"
	cfg.Paths.SearchRoots = nil
	if !startupBootstrapRequired(cfg) {
		t.Fatal("expected missing search roots to require startup bootstrap")
	}

	cfg.Identity.DisplayName = "Lane User"
	cfg.Paths.SearchRoots = []string{"/tmp/code"}
	if startupBootstrapRequired(cfg) {
		t.Fatal("expected complete identity + search roots to bypass startup bootstrap")
	}
}

// TestSanitizeBootstrapActorType verifies actor type normalization for bootstrap persistence.
func TestSanitizeBootstrapActorType(t *testing.T) {
	cases := map[string]string{
		"user":        "user",
		"AGENT":       "agent",
		" system ":    "system",
		"unexpected":  "user",
		"":            "user",
		"\nunknown\t": "user",
	}
	for input, want := range cases {
		if got := sanitizeBootstrapActorType(input); got != want {
			t.Fatalf("sanitizeBootstrapActorType(%q) = %q, want %q", input, got, want)
		}
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
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
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

// TestRunTUIModeWritesRuntimeLogsToFileOnly verifies TUI runtime logs stay out of stderr and persist to the dev log file.
func TestRunTUIModeWritesRuntimeLogsToFileOnly(t *testing.T) {
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })
	programFactory = func(_ tea.Model) program { return fakeProgram{} }

	workspace := t.TempDir()
	t.Chdir(workspace)

	dbPath := filepath.Join(workspace, "kan.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
	var stderr bytes.Buffer
	if err := run(context.Background(), []string{"--dev", "--db", dbPath, "--config", cfgPath}, io.Discard, &stderr); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if got := strings.TrimSpace(stderr.String()); got != "" {
		t.Fatalf("expected no runtime stderr output in TUI mode, got %q", got)
	}

	logDir := filepath.Join(workspace, ".kan", "log")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	var logPath string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		logPath = filepath.Join(logDir, entry.Name())
		break
	}
	if logPath == "" {
		t.Fatalf("expected a .log file in %s", logDir)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	logOutput := string(content)
	if !strings.Contains(logOutput, "starting tui program loop") {
		t.Fatalf("expected runtime log file to include TUI lifecycle entries, got %q", logOutput)
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

[identity]
display_name = "Lane User"
default_actor_type = "agent"

[paths]
search_roots = ["/tmp/code", "/tmp/docs"]

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
	wantSearchRootCode := filepath.Clean("/tmp/code")
	wantSearchRootDocs := filepath.Clean("/tmp/docs")
	if len(runtimeCfg.SearchRoots) != 2 || runtimeCfg.SearchRoots[0] != wantSearchRootCode || runtimeCfg.SearchRoots[1] != wantSearchRootDocs {
		t.Fatalf("unexpected search roots runtime config %#v", runtimeCfg.SearchRoots)
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
	if got := runtimeCfg.Identity.DisplayName; got != "Lane User" {
		t.Fatalf("expected identity display name Lane User, got %q", got)
	}
	if got := runtimeCfg.Identity.DefaultActorType; got != "agent" {
		t.Fatalf("expected identity actor type agent, got %q", got)
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

// TestPersistIdentityRoundTrip verifies behavior for the covered scenario.
func TestPersistIdentityRoundTrip(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "kan.toml")

	if err := persistIdentity(cfgPath, "Lane User", "agent"); err != nil {
		t.Fatalf("persistIdentity() error = %v", err)
	}
	cfg, err := config.Load(cfgPath, config.Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.Identity.DisplayName; got != "Lane User" {
		t.Fatalf("expected persisted identity display name Lane User, got %q", got)
	}
	if got := cfg.Identity.DefaultActorType; got != "agent" {
		t.Fatalf("expected persisted identity actor type agent, got %q", got)
	}
}

// TestPersistSearchRootsRoundTrip verifies behavior for the covered scenario.
func TestPersistSearchRootsRoundTrip(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "kan.toml")

	if err := persistSearchRoots(cfgPath, []string{"/tmp/code", "/tmp/docs", "/tmp/code"}); err != nil {
		t.Fatalf("persistSearchRoots() error = %v", err)
	}
	cfg, err := config.Load(cfgPath, config.Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	wantSearchRootCode := filepath.Clean("/tmp/code")
	wantSearchRootDocs := filepath.Clean("/tmp/docs")
	if len(cfg.Paths.SearchRoots) != 2 || cfg.Paths.SearchRoots[0] != wantSearchRootCode || cfg.Paths.SearchRoots[1] != wantSearchRootDocs {
		t.Fatalf("unexpected persisted search roots %#v", cfg.Paths.SearchRoots)
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

// TestRuntimeLoggerCanMuteConsoleSink verifies console output can be suppressed while other sinks remain active.
func TestRuntimeLoggerCanMuteConsoleSink(t *testing.T) {
	var console bytes.Buffer
	cfg := config.Default("/tmp/kan.db").Logging

	logger, err := newRuntimeLogger(&console, "kan", false, cfg, func() time.Time {
		return time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("newRuntimeLogger() error = %v", err)
	}

	logger.Info("before")
	logger.SetConsoleEnabled(false)
	logger.Info("during")
	logger.SetConsoleEnabled(true)
	logger.Info("after")

	out := console.String()
	if !strings.Contains(out, "before") {
		t.Fatalf("expected console log to include 'before', got %q", out)
	}
	if strings.Contains(out, "during") {
		t.Fatalf("expected muted console log to omit 'during', got %q", out)
	}
	if !strings.Contains(out, "after") {
		t.Fatalf("expected console log to include 'after', got %q", out)
	}
}
