package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// Config represents the persistent application configuration.
type Config struct {
	APIKey  string `json:"apiKey,omitempty"`
	BaseURL string `json:"baseUrl,omitempty"`
	Model   string `json:"model,omitempty"`
	LanIP   string `json:"lanIp,omitempty"`
}

const configFileName = "gtalk_config.json"

// configPath returns the full path to the config file (next to the executable).
func configPath() string {
	exe, err := os.Executable()
	if err != nil {
		return configFileName
	}
	return filepath.Join(filepath.Dir(exe), configFileName)
}

// LoadConfig reads the config from disk.
func LoadConfig() Config {
	var cfg Config
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg // file doesn't exist yet, return empty
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("⚠️  Config file parse error: %v", err)
		return Config{}
	}
	log.Printf("📁 Loaded config from %s", configPath())
	return cfg
}

// SaveConfig writes the config to disk.
func SaveConfig(cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(configPath(), data, 0600); err != nil {
		return err
	}
	log.Printf("💾 Config saved to %s", configPath())
	return nil
}
