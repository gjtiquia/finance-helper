package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type config struct {
	Server string `json:"server"`
}

var errConfigNotFound = errors.New("config not found")

var userConfigDir = os.UserConfigDir

func configPath() (string, error) {
	configDir, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("Could not determine config directory")
	}

	return filepath.Join(configDir, "finance-helper", "config.json"), nil
}

func saveConfig(cfg config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("Could not create config directory")
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Could not save config")
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("Could not save config")
	}

	return nil
}

func loadConfig() (config, string, error) {
	path, err := configPath()
	if err != nil {
		return config{}, "", err
	}

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config{}, path, errConfigNotFound
		}

		return config{}, path, fmt.Errorf("Could not read config")
	}
	defer file.Close()

	var cfg config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return config{}, path, fmt.Errorf("Could not parse config")
	}

	return cfg, path, nil
}
