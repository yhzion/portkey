package config

import (
	"fmt"
	"regexp"
)

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
