package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/evanschultz/kan/internal/adapters/storage/sqlite"
	"github.com/evanschultz/kan/internal/app"
	"github.com/evanschultz/kan/internal/config"
	"github.com/evanschultz/kan/internal/platform"
	"github.com/evanschultz/kan/internal/tui"
	"github.com/google/uuid"
)

// version stores a package-level helper value.
var version = "dev"

// program represents program data used by this package.
type program interface {
	Run() (tea.Model, error)
}

// programFactory stores a package-level helper value.
var programFactory = func(m tea.Model) program {
	return tea.NewProgram(m)
}

// main handles main.
func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// run runs the requested command flow.
func run(ctx context.Context, args []string, stdout, _ io.Writer) error {
	fs := flag.NewFlagSet("kan", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var (
		configPath string
		dbPath     string
		appName    string
		devMode    bool
		showVer    bool
	)
	defaultDevMode := version == "dev"
	if envDev, ok := parseBoolEnv("KAN_DEV_MODE"); ok {
		defaultDevMode = envDev
	}
	if envApp := strings.TrimSpace(os.Getenv("KAN_APP_NAME")); envApp != "" {
		appName = envApp
	} else {
		appName = "kan"
	}
	fs.StringVar(&configPath, "config", "", "path to config TOML")
	fs.StringVar(&dbPath, "db", "", "path to sqlite database")
	fs.StringVar(&appName, "app", appName, "application name for config/data path resolution")
	fs.BoolVar(&devMode, "dev", defaultDevMode, "use dev mode paths (<app>-dev)")
	fs.BoolVar(&showVer, "version", false, "show version")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if showVer {
		_, _ = fmt.Fprintf(stdout, "kan %s\n", version)
		return nil
	}

	paths, err := platform.DefaultPathsWithOptions(platform.Options{
		AppName: appName,
		DevMode: devMode,
	})
	if err != nil {
		return err
	}

	command := firstArg(fs.Args())
	switch command {
	case "paths":
		_, _ = fmt.Fprintf(stdout, "app: %s\n", appName)
		_, _ = fmt.Fprintf(stdout, "dev_mode: %t\n", devMode)
		_, _ = fmt.Fprintf(stdout, "config: %s\n", paths.ConfigPath)
		_, _ = fmt.Fprintf(stdout, "data_dir: %s\n", paths.DataDir)
		_, _ = fmt.Fprintf(stdout, "db: %s\n", paths.DBPath)
		return nil
	case "", "export", "import":
		// Continue.
	default:
		return fmt.Errorf("unknown command: %s", command)
	}

	dbOverridden := strings.TrimSpace(dbPath) != ""
	if configPath == "" {
		if envPath := strings.TrimSpace(os.Getenv("KAN_CONFIG")); envPath != "" {
			configPath = envPath
		} else {
			configPath = paths.ConfigPath
		}
	}
	if !dbOverridden {
		if envPath := strings.TrimSpace(os.Getenv("KAN_DB_PATH")); envPath != "" {
			dbPath = envPath
			dbOverridden = true
		} else {
			dbPath = paths.DBPath
		}
	}

	defaultCfg := config.Default(dbPath)
	cfg, err := config.Load(configPath, defaultCfg)
	if err != nil {
		return err
	}
	if dbOverridden {
		cfg.Database.Path = dbPath
	}

	repo, err := sqlite.Open(cfg.Database.Path)
	if err != nil {
		return err
	}
	defer func() {
		_ = repo.Close()
	}()

	svc := app.NewService(repo, uuid.NewString, nil, app.ServiceConfig{
		DefaultDeleteMode:        app.DeleteMode(cfg.Delete.DefaultMode),
		AutoCreateProjectColumns: true,
	})

	switch command {
	case "":
		// Fall through to the TUI.
	case "export":
		return runExport(ctx, svc, fs.Args()[1:], stdout)
	case "import":
		return runImport(ctx, svc, fs.Args()[1:])
	default:
		return fmt.Errorf("unknown command: %s", command)
	}

	if _, err := svc.EnsureDefaultProject(ctx); err != nil {
		return err
	}

	m := tui.NewModel(
		svc,
		tui.WithDefaultDeleteMode(app.DeleteMode(cfg.Delete.DefaultMode)),
		tui.WithTaskFieldConfig(tui.TaskFieldConfig{
			ShowPriority:    cfg.TaskFields.ShowPriority,
			ShowDueDate:     cfg.TaskFields.ShowDueDate,
			ShowLabels:      cfg.TaskFields.ShowLabels,
			ShowDescription: cfg.TaskFields.ShowDescription,
		}),
		tui.WithSearchConfig(tui.SearchConfig{
			CrossProject:    cfg.Search.CrossProject,
			IncludeArchived: cfg.Search.IncludeArchived,
			States:          append([]string(nil), cfg.Search.States...),
		}),
		tui.WithBoardConfig(tui.BoardConfig{
			ShowWIPWarnings: cfg.Board.ShowWIPWarnings,
			GroupBy:         cfg.Board.GroupBy,
		}),
		tui.WithConfirmConfig(tui.ConfirmConfig{
			Delete:     cfg.Confirm.Delete,
			Archive:    cfg.Confirm.Archive,
			HardDelete: cfg.Confirm.HardDelete,
			Restore:    cfg.Confirm.Restore,
		}),
		tui.WithUIConfig(tui.UIConfig{
			DueSoonWindows: cfg.DueSoonDurations(),
			ShowDueSummary: cfg.UI.ShowDueSummary,
		}),
		tui.WithLabelConfig(tui.LabelConfig{
			Global:         append([]string(nil), cfg.Labels.Global...),
			Projects:       cloneLabelProjectConfig(cfg.Labels.Projects),
			EnforceAllowed: cfg.Labels.EnforceAllowed,
		}),
		tui.WithProjectRoots(cloneProjectRoots(cfg.ProjectRoots)),
		tui.WithKeyConfig(tui.KeyConfig{
			CommandPalette: cfg.Keys.CommandPalette,
			QuickActions:   cfg.Keys.QuickActions,
			MultiSelect:    cfg.Keys.MultiSelect,
			ActivityLog:    cfg.Keys.ActivityLog,
			Undo:           cfg.Keys.Undo,
			Redo:           cfg.Keys.Redo,
		}),
	)
	_, err = programFactory(m).Run()
	return err
}

// runExport runs the requested command flow.
func runExport(ctx context.Context, svc *app.Service, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("kan export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var (
		outPath         string
		includeArchived bool
	)
	fs.StringVar(&outPath, "out", "-", "output file path ('-' for stdout)")
	fs.BoolVar(&includeArchived, "include-archived", true, "include archived projects/columns/tasks")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("unexpected export arguments: %v", fs.Args())
	}

	snap, err := svc.ExportSnapshot(ctx, includeArchived)
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')

	if outPath == "-" {
		_, err = stdout.Write(encoded)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, encoded, 0o644)
}

// runImport runs the requested command flow.
func runImport(ctx context.Context, svc *app.Service, args []string) error {
	fs := flag.NewFlagSet("kan import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var inPath string
	fs.StringVar(&inPath, "in", "", "input snapshot JSON file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("unexpected import arguments: %v", fs.Args())
	}
	if inPath == "" {
		return fmt.Errorf("--in is required")
	}

	content, err := os.ReadFile(inPath)
	if err != nil {
		return err
	}
	var snap app.Snapshot
	if err := json.Unmarshal(content, &snap); err != nil {
		return fmt.Errorf("decode snapshot json: %w", err)
	}
	return svc.ImportSnapshot(ctx, snap)
}

// firstArg handles first arg.
func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

// parseBoolEnv parses input into a normalized form.
func parseBoolEnv(name string) (bool, bool) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return false, false
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false
	}
	return v, true
}

// cloneLabelProjectConfig deep-copies per-project label lists.
func cloneLabelProjectConfig(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for key, labels := range in {
		out[key] = append([]string(nil), labels...)
	}
	return out
}

// cloneProjectRoots deep-copies project-root path mappings.
func cloneProjectRoots(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, path := range in {
		out[key] = path
	}
	return out
}
