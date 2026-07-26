package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// config holds the connection details needed to talk to the tapnote Worker.
// It's saved outside this project folder (in the OS user config directory),
// so the secret never ends up in this repo even if it's later put under
// version control.
type config struct {
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tapnoted", "config.json"), nil
}

func loadConfig() (config, error) {
	path, err := configPath()
	if err != nil {
		return config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, fmt.Errorf("no config found — run `tapnoted config set --url <url> --secret <secret>` first")
	}

	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func saveConfig(cfg config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}
