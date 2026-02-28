package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/fang"
	charmLog "github.com/charmbracelet/log"
	"github.com/google/uuid"
	serveradapter "github.com/hylla/hakoll/internal/adapters/server"
	servercommon "github.com/hylla/hakoll/internal/adapters/server/common"
	"github.com/hylla/hakoll/internal/adapters/storage/sqlite"
	"github.com/hylla/hakoll/internal/app"
	"github.com/hylla/hakoll/internal/config"
	"github.com/hylla/hakoll/internal/platform"
	"github.com/hylla/hakoll/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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

// serveCommandRunner starts the HTTP+MCP serve flow.
var serveCommandRunner = func(ctx context.Context, cfg serveradapter.Config, deps serveradapter.Dependencies) error {
	return serveradapter.Run(ctx, cfg, deps)
}

// supportsStyledOutputFunc allows tests to force styled output mode.
var supportsStyledOutputFunc = supportsStyledOutput

// main handles main.
func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}

// rootCommandOptions stores top-level CLI option values.
type rootCommandOptions struct {
	configPath  string
	dbPath      string
	appName     string
	devMode     bool
	showVersion bool
}

// serveCommandOptions stores serve subcommand option values.
type serveCommandOptions struct {
	httpBind    string
	apiEndpoint string
	mcpEndpoint string
}

// exportCommandOptions stores export subcommand option values.
type exportCommandOptions struct {
	outPath         string
	includeArchived bool
}

// importCommandOptions stores import subcommand option values.
type importCommandOptions struct {
	inPath string
}

// run executes the CLI command tree through Fang+Cobra.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	rootOpts := rootCommandOptions{
		appName: "hakoll",
		devMode: version == "dev",
	}
	if envDev, ok := parseBoolEnv("KOLL_DEV_MODE"); ok {
		rootOpts.devMode = envDev
	}
	if envApp := strings.TrimSpace(os.Getenv("KOLL_APP_NAME")); envApp != "" {
		rootOpts.appName = envApp
	}

	serveOpts := serveCommandOptions{
		httpBind:    "127.0.0.1:5437",
		apiEndpoint: "/api/v1",
		mcpEndpoint: "/mcp",
	}
	exportOpts := exportCommandOptions{
		outPath:         "-",
		includeArchived: true,
	}
	importOpts := importCommandOptions{}

	rootCmd := &cobra.Command{
		Use:           "koll",
		Short:         "Terminal kanban board with HTTP+MCP adapters",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeCommandFlow(cmd.Context(), "", rootOpts, serveOpts, exportOpts, importOpts, stdout, stderr)
		},
	}
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(args)

	rootCmd.PersistentFlags().StringVar(&rootOpts.configPath, "config", "", "Path to config TOML")
	rootCmd.PersistentFlags().StringVar(&rootOpts.dbPath, "db", "", "Path to sqlite database")
	rootCmd.PersistentFlags().StringVar(&rootOpts.appName, "app", rootOpts.appName, "Application name for config/data path resolution")
	rootCmd.PersistentFlags().BoolVar(&rootOpts.devMode, "dev", rootOpts.devMode, "Use dev mode paths (<app>-dev)")
	rootCmd.PersistentFlags().BoolVar(&rootOpts.showVersion, "version", false, "Show version")

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP and MCP endpoints",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeCommandFlow(cmd.Context(), "serve", rootOpts, serveOpts, exportOpts, importOpts, stdout, stderr)
		},
	}
	serveCmd.Flags().StringVar(&serveOpts.httpBind, "http", serveOpts.httpBind, "HTTP listen address")
	serveCmd.Flags().StringVar(&serveOpts.apiEndpoint, "api-endpoint", serveOpts.apiEndpoint, "HTTP API base endpoint")
	serveCmd.Flags().StringVar(&serveOpts.mcpEndpoint, "mcp-endpoint", serveOpts.mcpEndpoint, "MCP streamable HTTP endpoint")

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export a snapshot JSON payload",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeCommandFlow(cmd.Context(), "export", rootOpts, serveOpts, exportOpts, importOpts, stdout, stderr)
		},
	}
	exportCmd.Flags().StringVar(&exportOpts.outPath, "out", exportOpts.outPath, "Output file path ('-' for stdout)")
	exportCmd.Flags().BoolVar(&exportOpts.includeArchived, "include-archived", exportOpts.includeArchived, "Include archived projects/columns/tasks")

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import a snapshot JSON payload",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeCommandFlow(cmd.Context(), "import", rootOpts, serveOpts, exportOpts, importOpts, stdout, stderr)
		},
	}
	importCmd.Flags().StringVar(&importOpts.inPath, "in", "", "Input snapshot JSON file")

	pathsCmd := &cobra.Command{
		Use:   "paths",
		Short: "Print resolved config/data/db paths",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if rootOpts.showVersion {
				return writeVersion(stdout)
			}
			paths, err := platform.DefaultPathsWithOptions(platform.Options{
				AppName: rootOpts.appName,
				DevMode: rootOpts.devMode,
			})
			if err != nil {
				return err
			}
			return writePathsOutput(stdout, rootOpts, paths)
		},
	}

	rootCmd.AddCommand(serveCmd, exportCmd, importCmd, pathsCmd)
	return fang.Execute(
		ctx,
		rootCmd,
		fang.WithoutCompletions(),
		fang.WithoutManpage(),
		fang.WithoutVersion(),
	)
}

// writeVersion writes the current CLI version to stdout.
func writeVersion(stdout io.Writer) error {
	if _, err := fmt.Fprintf(stdout, "koll %s\n", version); err != nil {
		return fmt.Errorf("write version output: %w", err)
	}
	return nil
}

// writePathsOutput renders resolved paths using Fang-aligned styling.
func writePathsOutput(stdout io.Writer, opts rootCommandOptions, paths platform.Paths) error {
	if !supportsStyledOutputFunc(stdout) {
		return writePathsPlain(stdout, opts, paths)
	}

	isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	colors := fang.DefaultColorScheme(lipgloss.LightDark(isDark))
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(colors.Title)
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(colors.Flag)
	valueStyle := lipgloss.NewStyle().
		Foreground(colors.Description)

	rows := []struct {
		key   string
		value string
	}{
		{key: "app", value: opts.appName},
		{key: "dev_mode", value: fmt.Sprintf("%t", opts.devMode)},
		{key: "config", value: paths.ConfigPath},
		{key: "data_dir", value: paths.DataDir},
		{key: "db", value: paths.DBPath},
	}

	maxKeyWidth := 0
	for _, row := range rows {
		if len(row.key) > maxKeyWidth {
			maxKeyWidth = len(row.key)
		}
	}

	lines := make([]string, 0, len(rows)+1)
	lines = append(lines, titleStyle.Render("Resolved Paths"))
	for _, row := range rows {
		paddedKey := fmt.Sprintf("%-*s:", maxKeyWidth, row.key)
		line := lipgloss.JoinHorizontal(
			lipgloss.Left,
			keyStyle.Render(paddedKey),
			"  ",
			valueStyle.Render(row.value),
		)
		lines = append(lines, line)
	}
	if _, err := fmt.Fprintln(stdout, strings.Join(lines, "\n")); err != nil {
		return fmt.Errorf("write paths output: %w", err)
	}
	return nil
}

// writePathsPlain renders resolved paths in stable key/value text for scripts.
func writePathsPlain(stdout io.Writer, opts rootCommandOptions, paths platform.Paths) error {
	if _, err := fmt.Fprintf(stdout, "app: %s\n", opts.appName); err != nil {
		return fmt.Errorf("write paths app output: %w", err)
	}
	if _, err := fmt.Fprintf(stdout, "dev_mode: %t\n", opts.devMode); err != nil {
		return fmt.Errorf("write paths dev output: %w", err)
	}
	if _, err := fmt.Fprintf(stdout, "config: %s\n", paths.ConfigPath); err != nil {
		return fmt.Errorf("write paths config output: %w", err)
	}
	if _, err := fmt.Fprintf(stdout, "data_dir: %s\n", paths.DataDir); err != nil {
		return fmt.Errorf("write paths data output: %w", err)
	}
	if _, err := fmt.Fprintf(stdout, "db: %s\n", paths.DBPath); err != nil {
		return fmt.Errorf("write paths db output: %w", err)
	}
	return nil
}

// supportsStyledOutput reports whether output should include terminal styles.
func supportsStyledOutput(w io.Writer) bool {
	if strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

// executeCommandFlow runs the runtime setup + command-specific execution path.
func executeCommandFlow(
	ctx context.Context,
	command string,
	rootOpts rootCommandOptions,
	serveOpts serveCommandOptions,
	exportOpts exportCommandOptions,
	importOpts importCommandOptions,
	stdout io.Writer,
	stderr io.Writer,
) error {
	if rootOpts.showVersion {
		return writeVersion(stdout)
	}

	paths, err := platform.DefaultPathsWithOptions(platform.Options{
		AppName: rootOpts.appName,
		DevMode: rootOpts.devMode,
	})
	if err != nil {
		return err
	}

	configPath := rootOpts.configPath
	dbPath := rootOpts.dbPath
	dbOverridden := strings.TrimSpace(dbPath) != ""
	if configPath == "" {
		if envPath := strings.TrimSpace(os.Getenv("KOLL_CONFIG")); envPath != "" {
			configPath = envPath
		} else {
			configPath = paths.ConfigPath
		}
	}
	if !dbOverridden {
		if envPath := strings.TrimSpace(os.Getenv("KOLL_DB_PATH")); envPath != "" {
			dbPath = envPath
			dbOverridden = true
		} else {
			dbPath = paths.DBPath
		}
	}

	defaultCfg := config.Default(dbPath)
	cfg, err := config.Load(configPath, defaultCfg)
	if err != nil {
		return fmt.Errorf("load config %q: %w", configPath, err)
	}
	if dbOverridden {
		cfg.Database.Path = dbPath
	}
	bootstrapRequired := startupBootstrapRequired(cfg)

	logger, err := newRuntimeLogger(stderr, rootOpts.appName, rootOpts.devMode, cfg.Logging, time.Now)
	if err != nil {
		return fmt.Errorf("configure runtime logger: %w", err)
	}
	if command == "" {
		// Keep TUI rendering clean: runtime logs stay in the dev-file sink while the board is active.
		logger.SetConsoleEnabled(false)
	}
	defer func() {
		if closeErr := logger.Close(); closeErr != nil && logger.shouldLogToSink(logger.consoleSink) {
			// Keep TUI shutdown quiet on the terminal when console logging is intentionally muted.
			_, _ = fmt.Fprintf(stderr, "warning: close runtime log sink: %v\n", closeErr)
		}
	}()

	logger.Info("startup configuration resolved", "app", rootOpts.appName, "dev_mode", rootOpts.devMode, "command", command, "bootstrap_required", bootstrapRequired)
	logger.Debug("runtime paths resolved", "config_path", configPath, "data_dir", paths.DataDir, "db_path", dbPath)
	logger.Info("configuration loaded", "config_path", configPath, "db_path", cfg.Database.Path, "log_level", cfg.Logging.Level)
	if devPath := logger.DevLogPath(); devPath != "" {
		logger.Info("dev file logging enabled", "path", devPath)
	}

	logger.Info("opening sqlite repository", "db_path", cfg.Database.Path)
	repo, err := sqlite.Open(cfg.Database.Path)
	if err != nil {
		logger.Error("sqlite open failed", "db_path", cfg.Database.Path, "err", err)
		return fmt.Errorf("open sqlite repository: %w", err)
	}
	defer func() {
		if closeErr := repo.Close(); closeErr != nil {
			logger.Warn("sqlite close failed", "db_path", cfg.Database.Path, "err", closeErr)
		}
	}()
	logger.Info("sqlite repository ready", "db_path", cfg.Database.Path, "migrations", "ensured")

	svc := app.NewService(repo, uuid.NewString, nil, app.ServiceConfig{
		DefaultDeleteMode:        app.DeleteMode(cfg.Delete.DefaultMode),
		AutoCreateProjectColumns: true,
	})
	logger.Debug("application service initialized", "default_delete_mode", cfg.Delete.DefaultMode)

	switch command {
	case "":
		logger.Info("command flow start", "command", "tui")
	case "serve":
		logger.Info("command flow start", "command", "serve")
		if err := runServe(ctx, svc, rootOpts.appName, serveOpts); err != nil {
			logger.Error("command flow failed", "command", "serve", "err", err)
			return fmt.Errorf("run serve command: %w", err)
		}
		logger.Info("command flow complete", "command", "serve")
		return nil
	case "export":
		logger.Info("command flow start", "command", "export")
		if err := runExport(ctx, svc, exportOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "export", "err", err)
			return fmt.Errorf("run export command: %w", err)
		}
		logger.Info("command flow complete", "command", "export")
		return nil
	case "import":
		logger.Info("command flow start", "command", "import")
		if err := runImport(ctx, svc, importOpts); err != nil {
			logger.Error("command flow failed", "command", "import", "err", err)
			return fmt.Errorf("run import command: %w", err)
		}
		logger.Info("command flow complete", "command", "import")
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}

	m := tui.NewModel(
		svc,
		tui.WithLaunchProjectPicker(true),
		tui.WithStartupBootstrap(bootstrapRequired),
		tui.WithRuntimeConfig(toTUIRuntimeConfig(cfg)),
		tui.WithReloadConfigCallback(func() (tui.RuntimeConfig, error) {
			logger.Info("runtime config reload requested", "config_path", configPath)
			reloaded, err := loadRuntimeConfig(configPath, defaultCfg, dbPath, dbOverridden)
			if err != nil {
				logger.Error("runtime config reload failed", "config_path", configPath, "err", err)
				return tui.RuntimeConfig{}, err
			}
			logger.Info("runtime config reload complete", "config_path", configPath)
			return reloaded, nil
		}),
		tui.WithSaveProjectRootCallback(func(projectSlug, rootPath string) error {
			logger.Info("project root update requested", "project_slug", projectSlug, "root_path", rootPath, "config_path", configPath)
			if err := persistProjectRoot(configPath, projectSlug, rootPath); err != nil {
				logger.Error("project root update failed", "project_slug", projectSlug, "root_path", rootPath, "config_path", configPath, "err", err)
				return err
			}
			logger.Info("project root update complete", "project_slug", projectSlug, "root_path", rootPath, "config_path", configPath)
			return nil
		}),
		tui.WithSaveLabelsConfigCallback(func(projectSlug string, globalLabels, projectLabels []string) error {
			logger.Info("labels config update requested", "project_slug", projectSlug, "global_count", len(globalLabels), "project_count", len(projectLabels), "config_path", configPath)
			if err := persistAllowedLabels(configPath, projectSlug, globalLabels, projectLabels); err != nil {
				logger.Error("labels config update failed", "project_slug", projectSlug, "config_path", configPath, "err", err)
				return err
			}
			logger.Info("labels config update complete", "project_slug", projectSlug, "global_count", len(globalLabels), "project_count", len(projectLabels), "config_path", configPath)
			return nil
		}),
		tui.WithSaveBootstrapConfigCallback(func(cfg tui.BootstrapConfig) error {
			displayName := strings.TrimSpace(cfg.DisplayName)
			actorType := sanitizeBootstrapActorType(cfg.DefaultActorType)
			searchRoots := cloneSearchRoots(cfg.SearchRoots)
			logger.Info("bootstrap settings update requested", "config_path", configPath, "display_name", displayName, "default_actor_type", actorType, "search_roots_count", len(searchRoots))
			if err := persistIdentity(configPath, displayName, actorType); err != nil {
				logger.Error("bootstrap identity update failed", "config_path", configPath, "display_name", displayName, "default_actor_type", actorType, "err", err)
				return err
			}
			if err := persistSearchRoots(configPath, searchRoots); err != nil {
				logger.Error("bootstrap search roots update failed", "config_path", configPath, "search_roots_count", len(searchRoots), "err", err)
				return err
			}
			logger.Info("bootstrap settings update complete", "config_path", configPath, "display_name", displayName, "default_actor_type", actorType, "search_roots_count", len(searchRoots))
			return nil
		}),
	)
	logger.Info("starting tui program loop")
	_, err = programFactory(m).Run()
	if err != nil {
		logger.Error("tui program terminated with error", "err", err)
		return fmt.Errorf("run tui program: %w", err)
	}
	logger.Info("command flow complete", "command", "tui")
	return nil
}

// runServe runs the serve subcommand flow.
func runServe(ctx context.Context, svc *app.Service, appName string, opts serveCommandOptions) error {
	appAdapter := servercommon.NewAppServiceAdapter(svc)
	return serveCommandRunner(ctx, serveradapter.Config{
		HTTPBind:      opts.httpBind,
		APIEndpoint:   opts.apiEndpoint,
		MCPEndpoint:   opts.mcpEndpoint,
		ServerName:    appName,
		ServerVersion: version,
	}, serveradapter.Dependencies{
		CaptureState: appAdapter,
		Attention:    appAdapter,
	})
}

// runExport runs the requested command flow.
func runExport(ctx context.Context, svc *app.Service, opts exportCommandOptions, stdout io.Writer) error {
	snap, err := svc.ExportSnapshot(ctx, opts.includeArchived)
	if err != nil {
		return fmt.Errorf("export snapshot: %w", err)
	}
	encoded, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("encode snapshot json: %w", err)
	}
	encoded = append(encoded, '\n')

	if opts.outPath == "-" {
		if _, err := stdout.Write(encoded); err != nil {
			return fmt.Errorf("write snapshot to stdout: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(opts.outPath), 0o755); err != nil {
		return fmt.Errorf("create export output dir: %w", err)
	}
	if err := os.WriteFile(opts.outPath, encoded, 0o644); err != nil {
		return fmt.Errorf("write export file: %w", err)
	}
	return nil
}

// runImport runs the requested command flow.
func runImport(ctx context.Context, svc *app.Service, opts importCommandOptions) error {
	if opts.inPath == "" {
		return fmt.Errorf("--in is required")
	}

	content, err := os.ReadFile(opts.inPath)
	if err != nil {
		return fmt.Errorf("read import file: %w", err)
	}
	var snap app.Snapshot
	if err := json.Unmarshal(content, &snap); err != nil {
		return fmt.Errorf("decode snapshot json: %w", err)
	}
	if err := svc.ImportSnapshot(ctx, snap); err != nil {
		return fmt.Errorf("import snapshot: %w", err)
	}
	return nil
}

// startupBootstrapRequired reports whether startup must collect required identity/root settings in TUI.
func startupBootstrapRequired(cfg config.Config) bool {
	if strings.TrimSpace(cfg.Identity.DisplayName) == "" {
		return true
	}
	return len(cfg.Paths.SearchRoots) == 0
}

// sanitizeBootstrapActorType normalizes bootstrap actor type values to supported options.
func sanitizeBootstrapActorType(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "user", "agent", "system":
		return strings.TrimSpace(strings.ToLower(raw))
	default:
		return "user"
	}
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

// loadRuntimeConfig loads runtime-configurable options from disk.
func loadRuntimeConfig(configPath string, defaults config.Config, dbPath string, dbOverridden bool) (tui.RuntimeConfig, error) {
	cfg, err := config.Load(configPath, defaults)
	if err != nil {
		return tui.RuntimeConfig{}, fmt.Errorf("load config %q: %w", configPath, err)
	}
	if dbOverridden {
		cfg.Database.Path = dbPath
	}
	return toTUIRuntimeConfig(cfg), nil
}

// toTUIRuntimeConfig maps persisted config values into runtime model options.
func toTUIRuntimeConfig(cfg config.Config) tui.RuntimeConfig {
	return tui.RuntimeConfig{
		DefaultDeleteMode: app.DeleteMode(cfg.Delete.DefaultMode),
		TaskFields: tui.TaskFieldConfig{
			ShowPriority:    cfg.TaskFields.ShowPriority,
			ShowDueDate:     cfg.TaskFields.ShowDueDate,
			ShowLabels:      cfg.TaskFields.ShowLabels,
			ShowDescription: cfg.TaskFields.ShowDescription,
		},
		Search: tui.SearchConfig{
			CrossProject:    cfg.Search.CrossProject,
			IncludeArchived: cfg.Search.IncludeArchived,
			States:          append([]string(nil), cfg.Search.States...),
		},
		SearchRoots: cloneSearchRoots(cfg.Paths.SearchRoots),
		Confirm: tui.ConfirmConfig{
			Delete:     cfg.Confirm.Delete,
			Archive:    cfg.Confirm.Archive,
			HardDelete: cfg.Confirm.HardDelete,
			Restore:    cfg.Confirm.Restore,
		},
		Board: tui.BoardConfig{
			ShowWIPWarnings: cfg.Board.ShowWIPWarnings,
			GroupBy:         cfg.Board.GroupBy,
		},
		UI: tui.UIConfig{
			DueSoonWindows: cfg.DueSoonDurations(),
			ShowDueSummary: cfg.UI.ShowDueSummary,
		},
		Labels: tui.LabelConfig{
			Global:         append([]string(nil), cfg.Labels.Global...),
			Projects:       cloneLabelProjectConfig(cfg.Labels.Projects),
			EnforceAllowed: cfg.Labels.EnforceAllowed,
		},
		ProjectRoots: cloneProjectRoots(cfg.ProjectRoots),
		Keys: tui.KeyConfig{
			CommandPalette: cfg.Keys.CommandPalette,
			QuickActions:   cfg.Keys.QuickActions,
			MultiSelect:    cfg.Keys.MultiSelect,
			ActivityLog:    cfg.Keys.ActivityLog,
			Undo:           cfg.Keys.Undo,
			Redo:           cfg.Keys.Redo,
		},
		Identity: tui.IdentityConfig{
			DisplayName:      cfg.Identity.DisplayName,
			DefaultActorType: cfg.Identity.DefaultActorType,
		},
	}
}

// persistProjectRoot updates one project-root mapping in the TOML config file.
func persistProjectRoot(configPath, projectSlug, rootPath string) error {
	if err := config.UpsertProjectRoot(configPath, projectSlug, rootPath); err != nil {
		return fmt.Errorf("persist project root: %w", err)
	}
	return nil
}

// persistIdentity updates identity defaults in the TOML config file.
func persistIdentity(configPath, displayName, defaultActorType string) error {
	if err := config.UpsertIdentity(configPath, displayName, defaultActorType); err != nil {
		return fmt.Errorf("persist identity config: %w", err)
	}
	return nil
}

// persistSearchRoots updates global search roots in the TOML config file.
func persistSearchRoots(configPath string, searchRoots []string) error {
	if err := config.UpsertSearchRoots(configPath, searchRoots); err != nil {
		return fmt.Errorf("persist search roots config: %w", err)
	}
	return nil
}

// persistAllowedLabels updates global + project label defaults in the TOML config file.
func persistAllowedLabels(configPath, projectSlug string, globalLabels, projectLabels []string) error {
	if err := config.UpsertAllowedLabels(configPath, projectSlug, globalLabels, projectLabels); err != nil {
		return fmt.Errorf("persist labels config: %w", err)
	}
	return nil
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

// cloneSearchRoots deep-copies global search-root paths.
func cloneSearchRoots(in []string) []string {
	return append([]string(nil), in...)
}

// runtimeLogger fans log events to a styled console sink and an optional dev-file sink.
type runtimeLogger struct {
	sinks          []*charmLog.Logger
	consoleSink    *charmLog.Logger
	consoleEnabled bool
	closeFile      func() error
	devLog         string
}

// newRuntimeLogger configures runtime log sinks from CLI/config state.
func newRuntimeLogger(stderr io.Writer, appName string, devMode bool, cfg config.LoggingConfig, now func() time.Time) (*runtimeLogger, error) {
	level, err := charmLog.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("parse logging level %q: %w", cfg.Level, err)
	}

	if now == nil {
		now = time.Now
	}
	if stderr == nil {
		stderr = io.Discard
	}

	consoleLogger := charmLog.NewWithOptions(stderr, charmLog.Options{
		Level:           level,
		Prefix:          appName,
		ReportTimestamp: true,
		TimeFormat:      time.RFC3339,
		Formatter:       charmLog.TextFormatter,
	})

	logger := &runtimeLogger{
		sinks:          []*charmLog.Logger{consoleLogger},
		consoleSink:    consoleLogger,
		consoleEnabled: true,
	}
	if !devMode || !cfg.DevFile.Enabled {
		return logger, nil
	}

	devLogPath, err := devLogFilePath(cfg.DevFile.Dir, appName, now().UTC())
	if err != nil {
		return nil, fmt.Errorf("resolve dev log file path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(devLogPath), 0o755); err != nil {
		return nil, fmt.Errorf("create dev log dir: %w", err)
	}
	logFile, err := os.OpenFile(devLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open dev log file: %w", err)
	}

	// Keep file output parseable and unstyled while preserving styled console logs.
	fileLogger := charmLog.NewWithOptions(logFile, charmLog.Options{
		Level:           level,
		Prefix:          appName,
		ReportTimestamp: true,
		TimeFormat:      time.RFC3339,
		Formatter:       charmLog.LogfmtFormatter,
	})
	logger.sinks = append(logger.sinks, fileLogger)
	logger.closeFile = logFile.Close
	logger.devLog = devLogPath
	return logger, nil
}

// DevLogPath returns the active dev log file path.
func (l *runtimeLogger) DevLogPath() string {
	if l == nil {
		return ""
	}
	return l.devLog
}

// Close closes the optional dev-file sink.
func (l *runtimeLogger) Close() error {
	if l == nil || l.closeFile == nil {
		return nil
	}
	return l.closeFile()
}

// SetConsoleEnabled toggles whether the console sink receives runtime events.
func (l *runtimeLogger) SetConsoleEnabled(enabled bool) {
	if l == nil {
		return
	}
	l.consoleEnabled = enabled
}

// shouldLogToSink reports whether one sink should receive runtime output.
func (l *runtimeLogger) shouldLogToSink(sink *charmLog.Logger) bool {
	if l == nil {
		return false
	}
	if sink == nil {
		return false
	}
	if sink == l.consoleSink && !l.consoleEnabled {
		return false
	}
	return true
}

// Debug logs a debug event to all configured sinks.
func (l *runtimeLogger) Debug(msg string, keyvals ...any) {
	if l == nil {
		return
	}
	for _, sink := range l.sinks {
		if !l.shouldLogToSink(sink) {
			continue
		}
		sink.Debug(msg, keyvals...)
	}
}

// Info logs an informational event to all configured sinks.
func (l *runtimeLogger) Info(msg string, keyvals ...any) {
	if l == nil {
		return
	}
	for _, sink := range l.sinks {
		if !l.shouldLogToSink(sink) {
			continue
		}
		sink.Info(msg, keyvals...)
	}
}

// Warn logs a warning event to all configured sinks.
func (l *runtimeLogger) Warn(msg string, keyvals ...any) {
	if l == nil {
		return
	}
	for _, sink := range l.sinks {
		if !l.shouldLogToSink(sink) {
			continue
		}
		sink.Warn(msg, keyvals...)
	}
}

// Error logs an error event to all configured sinks.
func (l *runtimeLogger) Error(msg string, keyvals ...any) {
	if l == nil {
		return
	}
	for _, sink := range l.sinks {
		if !l.shouldLogToSink(sink) {
			continue
		}
		sink.Error(msg, keyvals...)
	}
}

// devLogFilePath resolves a workspace-local dev log file path for the current run day.
func devLogFilePath(configDir, appName string, now time.Time) (string, error) {
	baseDir := strings.TrimSpace(configDir)
	if baseDir == "" {
		baseDir = ".hakoll/log"
	}
	if !filepath.IsAbs(baseDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve working dir: %w", err)
		}
		baseDir = filepath.Join(workspaceRootFrom(cwd), baseDir)
	}
	fileStem := sanitizeLogFileStem(appName)
	fileName := fmt.Sprintf("%s-%s.log", fileStem, now.Format("20060102"))
	return filepath.Join(filepath.Clean(baseDir), fileName), nil
}

// workspaceRootFrom resolves the nearest ancestor workspace marker for stable local log placement.
func workspaceRootFrom(start string) string {
	start = filepath.Clean(strings.TrimSpace(start))
	if start == "" {
		return "."
	}
	dir := start
	for {
		if hasWorkspaceMarker(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return start
		}
		dir = parent
	}
}

// hasWorkspaceMarker reports whether a directory looks like a project workspace root.
func hasWorkspaceMarker(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return true
	}
	return false
}

// sanitizeLogFileStem normalizes app names into safe file-name segments.
func sanitizeLogFileStem(appName string) string {
	stem := strings.TrimSpace(appName)
	if stem == "" {
		return "hakoll"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
	stem = strings.Trim(replacer.Replace(stem), "-")
	if stem == "" {
		return "hakoll"
	}
	return stem
}
