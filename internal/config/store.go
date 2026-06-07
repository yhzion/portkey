package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Store is the interface for config persistence.
type Store interface {
	Load() (*Config, error)
	Save(cfg *Config) error
}

// FileStore implements Store with file-based I/O at an explicit path.
type FileStore struct {
	path string
}

// NewStore returns a Store backed by the file at path.
func NewStore(path string) Store {
	return &FileStore{path: path}
}

func (s *FileStore) Load() (*Config, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{Hosts: []Host{}}, nil
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	if len(data) == 0 {
		return &Config{Hosts: []Host{}}, nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	return &cfg, nil
}

func (s *FileStore) Save(cfg *Config) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// ConfigDir returns the portkey config directory path.
func ConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}
	return filepath.Join(dir, "portkey"), nil
}

// ConfigPath returns the full path to the hosts.json config file.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "hosts.json"), nil
}

// EnsureDir creates the config directory if it does not exist.
func EnsureDir() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

// Load reads config from the default path.
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	return NewStore(path).Load()
}

// Save writes config to the default path.
func Save(cfg *Config) error {
	if err := EnsureDir(); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return NewStore(path).Save(cfg)
}
