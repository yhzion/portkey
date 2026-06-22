package config

import (
	"fmt"
	"sort"
	"strings"
)

// FindHostByName finds a host by exact name match first, then by suffix match.
// Returns the index of the matched host.
// Returns an error if no match found, or if the suffix match is ambiguous.
//
// Callers that need to distinguish exact from suffix matches (e.g. to require
// confirmation before acting on a suffix match) should use FindHostByNameMatch.
// This wrapper discards the exact/suffix signal and is kept for commands that
// have not adopted it.
func (c *Config) FindHostByName(name string) (int, error) {
	idx, _, err := c.FindHostByNameMatch(name)
	return idx, err
}

// FindHostByNameMatch finds a host by exact name match first, then by suffix
// match. The boolean return is true when the match was exact (h.Name == name),
// and false when it was a suffix match (single host whose name ends with name).
// Returns an error if no match found, or if the suffix match is ambiguous (>=2
// hosts match). The exact/suffix distinction lets callers require explicit
// confirmation before acting on a fuzzy suffix match (issue #46).
func (c *Config) FindHostByNameMatch(name string) (int, bool, error) {
	// Exact match first
	for i, h := range c.Hosts {
		if h.Name == name {
			return i, true, nil
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
		return -1, false, fmt.Errorf("host %q not found", name)
	case 1:
		return matches[0], false, nil
	default:
		var names []string
		for _, idx := range matches {
			names = append(names, c.Hosts[idx].Name)
		}
		return -1, false, fmt.Errorf(
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
