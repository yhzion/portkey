package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Host struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
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

// nameRegex is the SSOT for valid host names: [a-z0-9_-] only.
var nameRegex = regexp.MustCompile(`^[a-z0-9_-]+$`)

// ValidateName checks that a name is non-empty and contains only [a-z0-9_-].
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if !nameRegex.MatchString(name) {
		return fmt.Errorf(
			"name %q contains invalid characters (allowed: lowercase a-z, 0-9, hyphen, underscore)",
			name,
		)
	}
	return nil
}

// IsDuplicateName returns true if another host (excluding excludeIndex) has the same name.
func (c *Config) IsDuplicateName(name string, excludeIndex int) bool {
	for i, h := range c.Hosts {
		if i == excludeIndex {
			continue
		}
		if h.Name == name {
			return true
		}
	}
	return false
}

// FindHostByName finds a host by exact name match first, then by suffix match.
// Returns the index of the matched host.
// Returns an error if no match found, or if the suffix match is ambiguous.
func (c *Config) FindHostByName(name string) (int, error) {
	// Exact match first
	for i, h := range c.Hosts {
		if h.Name == name {
			return i, nil
		}
	}

	// Suffix match: host name ends with the search term
	var matches []int
	for i, h := range c.Hosts {
		if strings.HasSuffix(h.Name, name) {
			matches = append(matches, i)
		}
	}

	switch len(matches) {
	case 0:
		return -1, fmt.Errorf("host %q not found", name)
	case 1:
		return matches[0], nil
	default:
		var names []string
		for _, idx := range matches {
			names = append(names, c.Hosts[idx].Name)
		}
		return -1, fmt.Errorf(
			"ambiguous name %q matches multiple hosts: %s",
			name,
			strings.Join(names, ", "),
		)
	}
}

// MigrateName converts a legacy display name to a valid slug.
// Lowercases, replaces spaces and dots with hyphens, strips invalid chars.
func MigrateName(displayName string, index int) string {
	name := strings.ToLower(displayName)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, ".", "-")
	// Strip any character not in [a-z0-9_-]
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	result := b.String()
	if result == "" {
		result = fmt.Sprintf("host-%d", index)
	}
	return result
}
