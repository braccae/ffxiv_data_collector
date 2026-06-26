package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type DatabaseConfig struct {
	Type string `json:"type"` // "sqlite", "postgres", "mysql", "mssql", "libsql"
	DSN  string `json:"dsn"`
}

type Config struct {
	Database     DatabaseConfig `json:"database"`
	WebSocketURL string         `json:"websocket_url"`
}

func getDefaultConfigPath(portable bool) (string, error) {
	if portable {
		return "config.json", nil
	}

	var configDir string
	if runtime.GOOS == "windows" {
		configDir = os.Getenv("LOCALAPPDATA")
		if configDir == "" {
			// Fallback if LOCALAPPDATA is not set
			appData := os.Getenv("APPDATA")
			if appData != "" {
				configDir = filepath.Join(appData, "Local")
			}
		}
	}

	if configDir == "" {
		var err error
		configDir, err = os.UserConfigDir()
		if err != nil {
			configDir = "."
		}
	}

	appDir := filepath.Join(configDir, "ffxiv_data_collector")
	err := os.MkdirAll(appDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create config directory: %v", err)
	}

	return filepath.Join(appDir, "config.json"), nil
}

func loadConfig(portable bool) (*Config, error) {
	configPath, err := getDefaultConfigPath(portable)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		defaultConfig := &Config{
			Database: DatabaseConfig{
				Type: "sqlite",
				DSN:  "ffxiv_events.db",
			},
			WebSocketURL: "ws://127.0.0.1:10501/ws",
		}

		// If not portable, change the DSN to absolute path in the config dir
		if !portable {
			defaultConfig.Database.DSN = filepath.Join(filepath.Dir(configPath), "ffxiv_events.db")
		}

		data, err := json.MarshalIndent(defaultConfig, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal default config: %v", err)
		}

		err = os.WriteFile(configPath, data, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write default config: %v", err)
		}

		applyEnvOverrides(defaultConfig)
		return defaultConfig, nil
	}

	// Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	applyEnvOverrides(&cfg)
	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if dbType := os.Getenv("FFXIV_DB_TYPE"); dbType != "" {
		cfg.Database.Type = dbType
	}
	if dbDsn := os.Getenv("FFXIV_DB_DSN"); dbDsn != "" {
		cfg.Database.DSN = dbDsn
	}
	if wsUrl := os.Getenv("FFXIV_WS_URL"); wsUrl != "" {
		cfg.WebSocketURL = wsUrl
	}
}
