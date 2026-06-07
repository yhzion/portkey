package tui

import (
	"testing"

	"github.com/yhzion/portkey/internal/config"
)

// --- FuzzyMatch core matching tests ---

func TestFuzzyMatch_ExactMatch(t *testing.T) {
	hosts := []config.Host{{Name: "dev", Username: "u", Host: "h", Port: 22}}
	results := FuzzyMatch(hosts, "dev")
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].HostIndex != 0 {
		t.Errorf("HostIndex = %d, want 0", results[0].HostIndex)
	}
}

func TestFuzzyMatch_SubstringMatch(t *testing.T) {
	hosts := []config.Host{{Name: "production-api", Username: "u", Host: "h", Port: 22}}
	results := FuzzyMatch(hosts, "prod")
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
}

func TestFuzzyMatch_FuzzyNonContiguous(t *testing.T) {
	hosts := []config.Host{{Name: "production-db", Username: "u", Host: "h", Port: 22}}
	results := FuzzyMatch(hosts, "pdb")
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1 for fuzzy 'pdb' matching 'production-db'", len(results))
	}
	// Verify positions: p=0, d=10, b=12 (in "production-db")
	positions := results[0].Positions
	if len(positions) != 3 {
		t.Fatalf("len(Positions) = %d, want 3", len(positions))
	}
	if positions[0] != 0 {
		t.Errorf("Positions[0] = %d, want 0 (p)", positions[0])
	}
}

func TestFuzzyMatch_CaseInsensitive(t *testing.T) {
	hosts := []config.Host{{Name: "Production-DB", Username: "u", Host: "h", Port: 22}}
	results := FuzzyMatch(hosts, "pdb")
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1 (case-insensitive)", len(results))
	}
}

func TestFuzzyMatch_NoMatch(t *testing.T) {
	hosts := []config.Host{{Name: "staging", Username: "u", Host: "h", Port: 22}}
	results := FuzzyMatch(hosts, "xyz")
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0 for no match", len(results))
	}
}

func TestFuzzyMatch_EmptyQuery(t *testing.T) {
	hosts := []config.Host{
		{Name: "dev", Username: "u", Host: "h", Port: 22},
		{Name: "prod", Username: "u", Host: "h", Port: 22},
	}
	results := FuzzyMatch(hosts, "")
	if len(results) != 2 {
		t.Errorf("len(results) = %d, want 2 (empty query returns all)", len(results))
	}
}

func TestFuzzyMatch_MatchesUsername(t *testing.T) {
	hosts := []config.Host{{Name: "server", Username: "admin", Host: "10.0.0.1", Port: 22}}
	results := FuzzyMatch(hosts, "adm")
	if len(results) != 1 {
		t.Errorf("len(results) = %d, want 1 (matched username)", len(results))
	}
}

func TestFuzzyMatch_MatchesHost(t *testing.T) {
	hosts := []config.Host{{Name: "server", Username: "u", Host: "192.168.1.100", Port: 22}}
	results := FuzzyMatch(hosts, "168")
	if len(results) != 1 {
		t.Errorf("len(results) = %d, want 1 (matched host address)", len(results))
	}
}

func TestFuzzyMatch_RankingOrder(t *testing.T) {
	hosts := []config.Host{
		{Name: "production-api", Username: "u", Host: "h1", Port: 22},
		{Name: "prod", Username: "u", Host: "h2", Port: 22},
		{Name: "api-production", Username: "u", Host: "h3", Port: 22},
	}
	results := FuzzyMatch(hosts, "prod")
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	// Exact name "prod" should rank highest (index 1)
	if results[0].HostIndex != 1 {
		t.Errorf("best match HostIndex = %d, want 1 (exact name 'prod')", results[0].HostIndex)
	}
}

func TestFuzzyMatch_ContiguousBetterThanFuzzy(t *testing.T) {
	hosts := []config.Host{
		{Name: "pdb", Username: "u", Host: "h1", Port: 22},           // exact, contiguous
		{Name: "production-db", Username: "u", Host: "h2", Port: 22}, // fuzzy
	}
	results := FuzzyMatch(hosts, "pdb")
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	// "pdb" (exact) should score higher than "production-db" (fuzzy)
	if results[0].HostIndex != 0 {
		t.Errorf("best match HostIndex = %d, want 0 (exact 'pdb' should rank higher)", results[0].HostIndex)
	}
}

func TestFuzzyMatch_WordBoundaryBonus(t *testing.T) {
	hosts := []config.Host{
		{Name: "api-prod", Username: "u", Host: "h1", Port: 22}, // p after hyphen
		{Name: "maprod", Username: "u", Host: "h2", Port: 22},   // p in middle
	}
	results := FuzzyMatch(hosts, "p")
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	// "api-prod" has p after word boundary, should rank higher than "maprod"
	if results[0].HostIndex != 0 {
		t.Errorf("best match HostIndex = %d, want 0 (word boundary bonus)", results[0].HostIndex)
	}
}

func TestFuzzyMatch_PositionsForHighlighting(t *testing.T) {
	hosts := []config.Host{{Name: "dev-server", Username: "u", Host: "h", Port: 22}}
	results := FuzzyMatch(hosts, "ds")
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	positions := results[0].Positions
	if len(positions) != 2 {
		t.Fatalf("len(Positions) = %d, want 2", len(positions))
	}
	// d=0, s=4 in "dev-server"
	if positions[0] != 0 {
		t.Errorf("Positions[0] = %d, want 0 (d)", positions[0])
	}
	if positions[1] != 4 {
		t.Errorf("Positions[1] = %d, want 4 (s)", positions[1])
	}
}

func TestFuzzyMatch_OutOfOrderNoMatch(t *testing.T) {
	hosts := []config.Host{{Name: "production-db", Username: "u", Host: "h", Port: 22}}
	results := FuzzyMatch(hosts, "dbp")
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0 (out of order)", len(results))
	}
}

func TestFuzzyMatch_SingleCharQuery(t *testing.T) {
	hosts := []config.Host{
		{Name: "alpha", Username: "u", Host: "h1", Port: 22},
		{Name: "beta", Username: "u", Host: "h2", Port: 22},
		{Name: "gamma", Username: "u", Host: "h3", Port: 22},
	}
	results := FuzzyMatch(hosts, "a")
	if len(results) != 3 {
		t.Errorf("len(results) = %d, want 3 (all contain 'a')", len(results))
	}
	// "alpha" has 'a' at start, should rank highest
	if results[0].HostIndex != 0 {
		t.Errorf("best match HostIndex = %d, want 0 (earliest 'a')", results[0].HostIndex)
	}
}

func TestFuzzyMatch_DuplicateResultsNotReturned(t *testing.T) {
	// A host whose name, username AND host all match should only appear once
	hosts := []config.Host{{Name: "admin", Username: "admin", Host: "admin.local", Port: 22}}
	results := FuzzyMatch(hosts, "adm")
	if len(results) != 1 {
		t.Errorf("len(results) = %d, want 1 (deduped)", len(results))
	}
}
