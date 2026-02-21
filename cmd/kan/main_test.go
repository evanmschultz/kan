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
	"github.com/evanschultz/kan/internal/domain"
)

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
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })
	programFactory = func(_ tea.Model) program { return fakeProgram{} }

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "kan.db")
	cfgPath := filepath.Join(tmp, "missing.toml")
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("initial run() error = %v", err)
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
	if snap.Version != app.SnapshotVersion {
		t.Fatalf("unexpected snapshot version %q", snap.Version)
	}
	if len(snap.Projects) == 0 {
		t.Fatal("expected at least one project in exported snapshot")
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
