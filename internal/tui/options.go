package tui

import "github.com/evanschultz/kan/internal/app"

// TaskFieldConfig holds configuration for task field.
type TaskFieldConfig struct {
	ShowPriority    bool
	ShowDueDate     bool
	ShowLabels      bool
	ShowDescription bool
}

// SearchConfig holds configuration for search.
type SearchConfig struct {
	CrossProject    bool
	IncludeArchived bool
	States          []string
}

// Option defines a functional option for model configuration.
type Option func(*Model)

// DefaultTaskFieldConfig returns default task field config.
func DefaultTaskFieldConfig() TaskFieldConfig {
	return TaskFieldConfig{
		ShowPriority:    true,
		ShowDueDate:     true,
		ShowLabels:      true,
		ShowDescription: false,
	}
}

// WithTaskFieldConfig returns an option that sets task field config.
func WithTaskFieldConfig(cfg TaskFieldConfig) Option {
	return func(m *Model) {
		m.taskFields = cfg
	}
}

// WithDefaultDeleteMode returns an option that sets default delete mode.
func WithDefaultDeleteMode(mode app.DeleteMode) Option {
	return func(m *Model) {
		switch mode {
		case app.DeleteModeArchive, app.DeleteModeHard:
			m.defaultDeleteMode = mode
		}
	}
}

// WithSearchConfig returns an option that sets search config.
func WithSearchConfig(cfg SearchConfig) Option {
	return func(m *Model) {
		m.searchCrossProject = cfg.CrossProject
		m.showArchived = cfg.IncludeArchived
		if len(cfg.States) > 0 {
			m.searchStates = append([]string(nil), cfg.States...)
		}
	}
}
