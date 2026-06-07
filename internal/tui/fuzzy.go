package tui

import (
	"strings"
	"unicode"

	"github.com/yhzion/portkey/internal/config"
)

// MatchResult holds a fuzzy match result with ranking and highlight positions.
type MatchResult struct {
	HostIndex int     // index in original []config.Host
	Score     float64 // higher is better
	Positions []int   // rune positions in the matched string for highlighting
}

// FuzzyMatch filters and ranks hosts by fuzzy-matching query against
// each host's Name, Username, and Host fields. Returns results sorted
// by descending score. An empty query returns all hosts with score 0.
func FuzzyMatch(hosts []config.Host, query string) []MatchResult {
	if query == "" {
		results := make([]MatchResult, len(hosts))
		for i := range hosts {
			results[i] = MatchResult{HostIndex: i, Score: 0}
		}
		return results
	}

	qLower := strings.ToLower(query)

	// Track best result per host index (dedup across multiple fields).
	bestForHost := make(map[int]MatchResult)

	for i, h := range hosts {
		candidates := []string{h.Name, h.Username, h.Host}
		for _, candidate := range candidates {
			result, ok := fuzzyString(strings.ToLower(candidate), qLower)
			if !ok {
				continue
			}
			result.HostIndex = i

			// Exact name match gets a big bonus.
			if strings.ToLower(h.Name) == qLower {
				result.Score += 100
			}

			if existing, exists := bestForHost[i]; !exists || result.Score > existing.Score {
				bestForHost[i] = result
			}
		}
	}

	results := make([]MatchResult, 0, len(bestForHost))
	for _, r := range bestForHost {
		results = append(results, r)
	}

	// Sort by descending score.
	sortByScore(results)
	return results
}

// fuzzyString performs fuzzy matching of query runes against target runes.
// Returns positions of matched characters and a score. Returns ok=false if
// the query doesn't match (characters not found in order).
func fuzzyString(target, query string) (MatchResult, bool) {
	targetRunes := []rune(target)
	queryRunes := []rune(query)

	// Collect all possible match position sets, pick the best scoring one.
	positions, score, ok := findBestMatch(targetRunes, queryRunes, 0, 0)
	if !ok {
		return MatchResult{}, false
	}

	return MatchResult{
		Score:     score,
		Positions: positions,
	}, true
}

// findBestMatch greedily finds the best match positions for query runes
// within target runes. Uses a forward scan with scoring.
func findBestMatch(
	target []rune,
	query []rune,
	tStart int,
	qStart int,
) ([]int, float64, bool) {
	if qStart >= len(query) {
		return nil, 0, true
	}
	if tStart >= len(target) {
		return nil, 0, false
	}

	var bestPositions []int
	bestScore := float64(-1)
	found := false

	for ti := tStart; ti <= len(target)-len(query)+qStart; ti++ {
		if target[ti] != query[qStart] {
			continue
		}

		restPositions, restScore, ok := findBestMatch(target, query, ti+1, qStart+1)
		if !ok {
			continue
		}

		found = true
		score := restScore

		// Word boundary bonus.
		if ti == 0 || isWordBoundary(target[ti-1]) {
			score += 10
		}

		// Contiguous bonus: if previous query char matched at ti-1.
		if len(restPositions) > 0 && restPositions[0] == ti+1 {
			score += 5
		}

		// Earlier match bonus (diminishing).
		score += float64(len(target)-ti) * 0.1

		if score > bestScore {
			bestScore = score
			bestPositions = append([]int{ti}, restPositions...)
		}
	}

	if !found {
		return nil, 0, false
	}

	return bestPositions, bestScore, true
}

// isWordBoundary returns true if the rune is a word separator.
func isWordBoundary(r rune) bool {
	return r == '-' || r == '_' || r == '.' || unicode.IsSpace(r)
}

// sortByScore sorts results by descending score using insertion sort
// (good enough for the small host lists we deal with).
func sortByScore(results []MatchResult) {
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Score > results[j-1].Score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}
