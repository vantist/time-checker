package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

var defaults = map[string]string{
	"idle-threshold": "15",
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tt", "config.json"), nil
}

func load() (map[string]string, error) {
	p, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]string{}, nil
	}
	return m, nil
}

func save(m map[string]string) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func Set(key, value string) error {
	m, err := load()
	if err != nil {
		return err
	}
	m[key] = value
	return save(m)
}

func Get(key string) (string, error) {
	m, err := load()
	if err != nil {
		return "", err
	}
	if v, ok := m[key]; ok {
		return v, nil
	}
	return defaults[key], nil
}
