package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

	return nil
}

func EnsureConfigDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
