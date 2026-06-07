package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Host struct {
	DisplayName string `json:"displayName"`
	Username    string `json:"username"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
}

type Config struct {
	Hosts []Host `json:"hosts"`
}

// Store handles file-based config I/O at an explicit path.
type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load() (*Config, error) {
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

func (s *Store) Save(cfg *Config) error {
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

func ConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}
	return filepath.Join(dir, "portkey"), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "hosts.json"), nil
}

func EnsureDir() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	return NewStore(path).Load()
}

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

func (c *Config) AddHost(h Host) {
	c.Hosts = append(c.Hosts, h)
}

func (c *Config) UpdateHost(index int, h Host) {
	if index >= 0 && index < len(c.Hosts) {
		c.Hosts[index] = h
	}
}

func (c *Config) RemoveHost(index int) {
	if index >= 0 && index < len(c.Hosts) {
		c.Hosts = append(c.Hosts[:index], c.Hosts[index+1:]...)
	}
}
