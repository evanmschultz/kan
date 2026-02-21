package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

type DeleteMode string

const (
	DeleteModeArchive DeleteMode = "archive"
	DeleteModeHard    DeleteMode = "hard"
)

type Config struct {
	Database   DatabaseConfig   `toml:"database"`
	Delete     DeleteConfig     `toml:"delete"`
	TaskFields TaskFieldsConfig `toml:"task_fields"`
	Board      BoardConfig      `toml:"board"`
	Search     SearchConfig     `toml:"search"`
	Keys       KeyConfig        `toml:"keys"`
}

type DatabaseConfig struct {
	Path string `toml:"path"`
}

type DeleteConfig struct {
	DefaultMode DeleteMode `toml:"default_mode"`
}

type TaskFieldsConfig struct {
	ShowPriority    bool `toml:"show_priority"`
	ShowDueDate     bool `toml:"show_due_date"`
	ShowLabels      bool `toml:"show_labels"`
	ShowDescription bool `toml:"show_description"`
}

type BoardConfig struct {
	States          []StateConfig `toml:"states"`
	ShowWIPWarnings bool          `toml:"show_wip_warnings"`
	GroupBy         string        `toml:"group_by"` // none | priority | state
}

type StateConfig struct {
	ID       string `toml:"id"`
	Name     string `toml:"name"`
	WIPLimit int    `toml:"wip_limit"`
	Position int    `toml:"position"`
}

type SearchConfig struct {
	CrossProject    bool     `toml:"cross_project"`
	IncludeArchived bool     `toml:"include_archived"`
	States          []string `toml:"states"`
}

type KeyConfig struct {
	CommandPalette string `toml:"command_palette"`
	QuickActions   string `toml:"quick_actions"`
	MultiSelect    string `toml:"multi_select"`
	ActivityLog    string `toml:"activity_log"`
	Undo           string `toml:"undo"`
	Redo           string `toml:"redo"`
}

func defaultStates() []StateConfig {
	return []StateConfig{
		{ID: "todo", Name: "To Do", WIPLimit: 0, Position: 0},
		{ID: "progress", Name: "In Progress", WIPLimit: 0, Position: 1},
		{ID: "done", Name: "Done", WIPLimit: 0, Position: 2},
	}
}

func Default(dbPath string) Config {
	return Config{
		Database: DatabaseConfig{
			Path: dbPath,
		},
		Delete: DeleteConfig{
			DefaultMode: DeleteModeArchive,
		},
		TaskFields: TaskFieldsConfig{
			ShowPriority:    true,
			ShowDueDate:     true,
			ShowLabels:      true,
			ShowDescription: false,
		},
		Board: BoardConfig{
			States:          defaultStates(),
			ShowWIPWarnings: true,
			GroupBy:         "none",
		},
		Search: SearchConfig{
			CrossProject:    false,
			IncludeArchived: false,
			States:          []string{"todo", "progress", "done"},
		},
		Keys: KeyConfig{
			CommandPalette: ":",
			QuickActions:   ".",
			MultiSelect:    " ",
			ActivityLog:    "g",
			Undo:           "z",
			Redo:           "Z",
		},
	}
}

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

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	c.Database.Path = strings.TrimSpace(c.Database.Path)
	if c.Database.Path == "" {
		return errors.New("database path is required")
	}

	switch c.Delete.DefaultMode {
	case DeleteModeArchive, DeleteModeHard:
	default:
		return fmt.Errorf("invalid delete.default_mode: %q", c.Delete.DefaultMode)
	}

	if len(c.Board.States) == 0 {
		return errors.New("board.states must include at least one state")
	}
	seenStateID := map[string]struct{}{}
	for idx := range c.Board.States {
		state := c.Board.States[idx]
		state.ID = strings.TrimSpace(strings.ToLower(state.ID))
		state.Name = strings.TrimSpace(state.Name)
		if state.ID == "" {
			return fmt.Errorf("board.states[%d].id is required", idx)
		}
		if state.Name == "" {
			return fmt.Errorf("board.states[%d].name is required", idx)
		}
		if state.WIPLimit < 0 {
			return fmt.Errorf("board.states[%d].wip_limit must be >= 0", idx)
		}
		if state.Position < 0 {
			return fmt.Errorf("board.states[%d].position must be >= 0", idx)
		}
		if _, ok := seenStateID[state.ID]; ok {
			return fmt.Errorf("board.states[%d].id is duplicated: %s", idx, state.ID)
		}
		seenStateID[state.ID] = struct{}{}
		c.Board.States[idx] = state
	}
	switch strings.TrimSpace(strings.ToLower(c.Board.GroupBy)) {
	case "", "none", "priority", "state":
	default:
		return fmt.Errorf("invalid board.group_by: %q", c.Board.GroupBy)
	}

	knownStates := make([]string, 0, len(c.Board.States)+1)
	for _, state := range c.Board.States {
		knownStates = append(knownStates, state.ID)
	}
	knownStates = append(knownStates, "archived")
	for i, state := range c.Search.States {
		s := strings.TrimSpace(strings.ToLower(state))
		if s == "" {
			continue
		}
		if !slices.Contains(knownStates, s) {
			return fmt.Errorf("search.states[%d] references unknown state %q", i, s)
		}
	}

	return nil
}

func EnsureConfigDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
