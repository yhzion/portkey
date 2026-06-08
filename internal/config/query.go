package config

import (
	"fmt"
	"sort"
	"strings"
)

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

// SortHosts sorts hosts by LastUsed descending. Hosts with empty LastUsed
// sort to the bottom, grouped alphabetically by Name.
func SortHosts(hosts []Host) {
	sort.SliceStable(hosts, func(i, j int) bool {
		if hosts[i].LastUsed != "" && hosts[j].LastUsed != "" {
			return hosts[i].LastUsed > hosts[j].LastUsed
		}
		if hosts[i].LastUsed != "" {
			return true
		}
		if hosts[j].LastUsed != "" {
			return false
		}
		return hosts[i].Name < hosts[j].Name
	})
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
