package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

// DeleteMode represents a selectable mode.
type DeleteMode string

// DeleteModeArchive and related constants define package defaults.
const (
	DeleteModeArchive DeleteMode = "archive"
	DeleteModeHard    DeleteMode = "hard"
)

// Config holds package configuration.
type Config struct {
	Database     DatabaseConfig    `toml:"database"`
	Delete       DeleteConfig      `toml:"delete"`
	Confirm      ConfirmConfig     `toml:"confirm"`
	TaskFields   TaskFieldsConfig  `toml:"task_fields"`
	Board        BoardConfig       `toml:"board"`
	Search       SearchConfig      `toml:"search"`
	UI           UIConfig          `toml:"ui"`
	ProjectRoots map[string]string `toml:"project_roots"`
	Labels       LabelConfig       `toml:"labels"`
	Keys         KeyConfig         `toml:"keys"`
}

// DatabaseConfig holds configuration for database.
type DatabaseConfig struct {
	Path string `toml:"path"`
}

// DeleteConfig holds configuration for delete.
type DeleteConfig struct {
	DefaultMode DeleteMode `toml:"default_mode"`
}

// ConfirmConfig holds configuration for confirmation behavior.
type ConfirmConfig struct {
	Delete     bool `toml:"delete"`
	Archive    bool `toml:"archive"`
	HardDelete bool `toml:"hard_delete"`
	Restore    bool `toml:"restore"`
}

// TaskFieldsConfig holds configuration for task fields.
type TaskFieldsConfig struct {
	ShowPriority    bool `toml:"show_priority"`
	ShowDueDate     bool `toml:"show_due_date"`
	ShowLabels      bool `toml:"show_labels"`
	ShowDescription bool `toml:"show_description"`
}

// BoardConfig holds configuration for board.
type BoardConfig struct {
	ShowWIPWarnings bool   `toml:"show_wip_warnings"`
	GroupBy         string `toml:"group_by"` // none | priority | state
}

// SearchConfig holds configuration for search.
type SearchConfig struct {
	CrossProject    bool     `toml:"cross_project"`
	IncludeArchived bool     `toml:"include_archived"`
	States          []string `toml:"states"`
}

// UIConfig holds configuration for UI behavior.
type UIConfig struct {
	DueSoonWindows []string `toml:"due_soon_windows"`
	ShowDueSummary bool     `toml:"show_due_summary"`
}

// LabelConfig holds label suggestion and enforcement configuration.
type LabelConfig struct {
	Global         []string            `toml:"global"`
	Projects       map[string][]string `toml:"projects"`
	EnforceAllowed bool                `toml:"enforce_allowed"`
}

// KeyConfig holds configuration for key.
type KeyConfig struct {
	CommandPalette string `toml:"command_palette"`
	QuickActions   string `toml:"quick_actions"`
	MultiSelect    string `toml:"multi_select"`
	ActivityLog    string `toml:"activity_log"`
	Undo           string `toml:"undo"`
	Redo           string `toml:"redo"`
}

// Default returns default the requested value.
func Default(dbPath string) Config {
	return Config{
		Database: DatabaseConfig{
			Path: dbPath,
		},
		Delete: DeleteConfig{
			DefaultMode: DeleteModeArchive,
		},
		Confirm: ConfirmConfig{
			Delete:     true,
			Archive:    true,
			HardDelete: true,
			Restore:    false,
		},
		TaskFields: TaskFieldsConfig{
			ShowPriority:    true,
			ShowDueDate:     true,
			ShowLabels:      true,
			ShowDescription: false,
		},
		Board: BoardConfig{
			ShowWIPWarnings: true,
			GroupBy:         "none",
		},
		Search: SearchConfig{
			CrossProject:    false,
			IncludeArchived: false,
			States:          []string{"todo", "progress", "done"},
		},
		UI: UIConfig{
			DueSoonWindows: []string{"24h", "1h"},
			ShowDueSummary: true,
		},
		ProjectRoots: map[string]string{},
		Labels: LabelConfig{
			Global:         []string{},
			Projects:       map[string][]string{},
			EnforceAllowed: false,
		},
		Keys: KeyConfig{
			CommandPalette: ":",
			QuickActions:   ".",
			MultiSelect:    "space",
			ActivityLog:    "g",
			Undo:           "z",
			Redo:           "Z",
		},
	}
}

// Load loads required data for the current operation.
func Load(path string, defaults Config) (Config, error) {
	cfg := defaults
	if strings.TrimSpace(path) == "" {
		return cfg, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	if len(content) == 0 {
		return cfg, nil
	}

	if err := toml.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode toml: %w", err)
	}
	cfg.normalize()

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate validates the requested operation.
func (c *Config) Validate() error {
	c.Database.Path = strings.TrimSpace(c.Database.Path)
	if c.Database.Path == "" {
		return errors.New("database path is required")
	}

	switch c.Delete.DefaultMode {
	case DeleteModeArchive, DeleteModeHard:
	default:
		return fmt.Errorf("invalid delete.default_mode: %q", c.Delete.DefaultMode)
	}

	switch strings.TrimSpace(strings.ToLower(c.Board.GroupBy)) {
	case "", "none", "priority", "state":
	default:
		return fmt.Errorf("invalid board.group_by: %q", c.Board.GroupBy)
	}

	for i, state := range c.Search.States {
		if !isKnownLifecycleState(state) {
			return fmt.Errorf("search.states[%d] references unknown state %q", i, state)
		}
	}

	for i, raw := range c.UI.DueSoonWindows {
		window := strings.TrimSpace(raw)
		if window == "" {
			continue
		}
		d, err := time.ParseDuration(window)
		if err != nil {
			return fmt.Errorf("ui.due_soon_windows[%d] invalid duration %q", i, raw)
		}
		if d <= 0 {
			return fmt.Errorf("ui.due_soon_windows[%d] must be > 0", i)
		}
	}
	for key, rootPath := range c.ProjectRoots {
		if strings.TrimSpace(key) == "" {
			return errors.New("project_roots contains an empty key")
		}
		if strings.TrimSpace(rootPath) == "" {
			return fmt.Errorf("project_roots.%s path is empty", key)
		}
	}
	for projectSlug, labels := range c.Labels.Projects {
		if strings.TrimSpace(projectSlug) == "" {
			return errors.New("labels.projects contains an empty project key")
		}
		for i, label := range labels {
			if strings.TrimSpace(label) == "" {
				return fmt.Errorf("labels.projects.%s[%d] is empty", projectSlug, i)
			}
		}
	}

	return nil
}

// DueSoonDurations handles due soon durations.
func (c Config) DueSoonDurations() []time.Duration {
	out := make([]time.Duration, 0, len(c.UI.DueSoonWindows))
	seen := map[time.Duration]struct{}{}
	for _, raw := range c.UI.DueSoonWindows {
		s := strings.TrimSpace(strings.ToLower(raw))
		if s == "" {
			continue
		}
		d, err := time.ParseDuration(s)
		if err != nil || d <= 0 {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})
	return out
}

// AllowedLabels returns normalized allowed label suggestions for a project slug.
func (c Config) AllowedLabels(projectSlug string) []string {
	projectSlug = strings.TrimSpace(strings.ToLower(projectSlug))
	out := make([]string, 0)
	seen := map[string]struct{}{}
	appendUnique := func(values []string) {
		for _, value := range values {
			label := strings.TrimSpace(strings.ToLower(value))
			if label == "" {
				continue
			}
			if _, ok := seen[label]; ok {
				continue
			}
			seen[label] = struct{}{}
			out = append(out, label)
		}
	}
	appendUnique(c.Labels.Global)
	if labels, ok := c.Labels.Projects[projectSlug]; ok {
		appendUnique(labels)
	}
	sort.Strings(out)
	return out
}

// normalize canonicalizes config slices/maps after defaults + TOML overlay.
func (c *Config) normalize() {
	states := make([]string, 0, len(c.Search.States))
	seenStates := map[string]struct{}{}
	for _, raw := range c.Search.States {
		state := strings.TrimSpace(strings.ToLower(raw))
		if state == "" {
			continue
		}
		if _, ok := seenStates[state]; ok {
			continue
		}
		seenStates[state] = struct{}{}
		states = append(states, state)
	}
	if len(states) == 0 {
		states = []string{"todo", "progress", "done"}
	}
	c.Search.States = states

	windows := make([]string, 0, len(c.UI.DueSoonWindows))
	seenWindows := map[string]struct{}{}
	for _, raw := range c.UI.DueSoonWindows {
		window := strings.TrimSpace(strings.ToLower(raw))
		if window == "" {
			continue
		}
		if _, ok := seenWindows[window]; ok {
			continue
		}
		seenWindows[window] = struct{}{}
		windows = append(windows, window)
	}
	if len(windows) == 0 {
		windows = []string{"24h", "1h"}
	}
	c.UI.DueSoonWindows = windows

	roots := make(map[string]string, len(c.ProjectRoots))
	for rawKey, rawPath := range c.ProjectRoots {
		key := strings.TrimSpace(strings.ToLower(rawKey))
		path := strings.TrimSpace(rawPath)
		if key == "" || path == "" {
			continue
		}
		roots[key] = path
	}
	c.ProjectRoots = roots

	c.Labels.Global = normalizeLabelConfigList(c.Labels.Global)
	projectLabels := make(map[string][]string, len(c.Labels.Projects))
	for rawKey, labels := range c.Labels.Projects {
		key := strings.TrimSpace(strings.ToLower(rawKey))
		if key == "" {
			continue
		}
		projectLabels[key] = normalizeLabelConfigList(labels)
	}
	c.Labels.Projects = projectLabels

	c.Keys.CommandPalette = normalizeKeyBinding(c.Keys.CommandPalette, ":")
	c.Keys.QuickActions = normalizeKeyBinding(c.Keys.QuickActions, ".")
	c.Keys.MultiSelect = normalizeKeyBinding(c.Keys.MultiSelect, "space")
	c.Keys.ActivityLog = normalizeKeyBinding(c.Keys.ActivityLog, "g")
	c.Keys.Undo = normalizeKeyBinding(c.Keys.Undo, "z")
	c.Keys.Redo = normalizeKeyBinding(c.Keys.Redo, "Z")
}

// normalizeLabelConfigList trims, lowercases, and deduplicates label config entries.
func normalizeLabelConfigList(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
		label := strings.TrimSpace(strings.ToLower(raw))
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		out = append(out, label)
	}
	sort.Strings(out)
	return out
}

// EnsureConfigDir ensures config dir.
func EnsureConfigDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

// isKnownLifecycleState reports whether the requested condition is satisfied.
func isKnownLifecycleState(state string) bool {
	return slices.Contains([]string{"todo", "progress", "done", "archived"}, state)
}

// normalizeKeyBinding trims keybinding text and applies fallback defaults.
func normalizeKeyBinding(raw, fallback string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}
	return value
}
