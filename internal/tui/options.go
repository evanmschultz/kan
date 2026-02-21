package tui

import "github.com/evanschultz/kan/internal/app"

type TaskFieldConfig struct {
	ShowPriority    bool
	ShowDueDate     bool
	ShowLabels      bool
	ShowDescription bool
}

type Option func(*Model)

func DefaultTaskFieldConfig() TaskFieldConfig {
	return TaskFieldConfig{
		ShowPriority:    true,
		ShowDueDate:     true,
		ShowLabels:      true,
		ShowDescription: false,
	}
}

func WithTaskFieldConfig(cfg TaskFieldConfig) Option {
	return func(m *Model) {
		m.taskFields = cfg
	}
}

func WithDefaultDeleteMode(mode app.DeleteMode) Option {
	return func(m *Model) {
		switch mode {
		case app.DeleteModeArchive, app.DeleteModeHard:
			m.defaultDeleteMode = mode
		}
	}
}
