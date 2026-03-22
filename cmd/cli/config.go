package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type config struct {
	Server string `json:"server"`
}

var errConfigNotFound = errors.New("config not found")

func configPath() (string, error) {
	configDir, err := configBaseDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "finance-helper", "config.json"), nil
}

func configBaseDir() (string, error) {
	if runtime.GOOS == "windows" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("Could not determine config directory")
		}

		return configDir, nil
	}

	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return xdgConfigHome, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("Could not determine home directory")
	}

	return filepath.Join(homeDir, ".config"), nil
}

func saveConfig(cfg config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	return saveConfigAtPath(path, cfg)
}

func saveConfigAtPath(path string, cfg config) error {

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

	cfg, err := loadConfigAtPath(path)
	return cfg, path, err
}

func loadConfigAtPath(path string) (config, error) {

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config{}, errConfigNotFound
		}

		return config{}, fmt.Errorf("Could not read config")
	}
	defer file.Close()

	var cfg config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return config{}, fmt.Errorf("Could not parse config")
	}

	return cfg, nil
}
