package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Paths struct {
	ConfigPath string
	DataDir    string
	DBPath     string
}

func DefaultPaths() (Paths, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("user config dir: %w", err)
	}
	dataDir, err := os.UserCacheDir()
	if err != nil {
		return Paths{}, fmt.Errorf("user data dir: %w", err)
	}

	env := map[string]string{
		"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"),
		"XDG_DATA_HOME":   os.Getenv("XDG_DATA_HOME"),
		"APPDATA":         os.Getenv("APPDATA"),
		"LOCALAPPDATA":    os.Getenv("LOCALAPPDATA"),
	}
	return PathsFor(runtime.GOOS, env, configDir, dataDir)
}

func PathsFor(goos string, env map[string]string, userConfigDir, userDataDir string) (Paths, error) {
	if userConfigDir == "" || userDataDir == "" {
		return Paths{}, fmt.Errorf("empty base dirs")
	}

	configBase := userConfigDir
	dataBase := userDataDir

	switch goos {
	case "linux":
		if v := env["XDG_CONFIG_HOME"]; v != "" {
			configBase = v
		}
		if v := env["XDG_DATA_HOME"]; v != "" {
			dataBase = v
		}
	case "windows":
		if v := env["APPDATA"]; v != "" {
			configBase = v
		}
		if v := env["LOCALAPPDATA"]; v != "" {
			dataBase = v
		}
	case "darwin":
		// Keep os.UserConfigDir/UserCacheDir defaults for macOS.
	default:
		// Fallback for other platforms.
	}

	appConfigDir := filepath.Join(configBase, "kan")
	appDataDir := filepath.Join(dataBase, "kan")
	return Paths{
		ConfigPath: filepath.Join(appConfigDir, "config.toml"),
		DataDir:    appDataDir,
		DBPath:     filepath.Join(appDataDir, "kan.db"),
	}, nil
}
