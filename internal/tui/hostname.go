package tui

// Host-name column sizing. The name column is sized to the longest name on
// screen (so it never wraps) but clamped to [nameColMin, nameColMax]; names
// longer than the column are truncated with an ellipsis rather than wrapped.
const (
	nameColMin = 12
	nameColMax = 32
)

// truncateName limits name to width display columns, appending "…" when it had
// to cut. It returns the (possibly truncated) text and kept — the number of
// original runes retained — so callers can drop match-highlight positions that
// fall in the cut-off region. Truncation is rune-aware so multi-byte names are
// never split mid-rune.
func truncateName(name string, width int) (string, int) {
	if width <= 0 {
		return "", 0
	}
	runes := []rune(name)
	if len(runes) <= width {
		return name, len(runes)
	}
	kept := width - 1
	return string(runes[:kept]) + "…", kept
}

// keepPositions returns the match-highlight positions that survive truncation —
// those with index < kept. A new slice is returned; the input is not mutated.
func keepPositions(positions []int, kept int) []int {
	out := make([]int, 0, len(positions))
	for _, p := range positions {
		if p < kept {
			out = append(out, p)
		}
	}
	return out
}

// nameColumnWidth returns the width of the name column: the longest name in
// names, clamped to [nameColMin, nameColMax]. Sizing to the content keeps the
// column tight while the cap bounds a single pathological name.
func nameColumnWidth(names []string) int {
	w := nameColMin
	for _, n := range names {
		if l := len([]rune(n)); l > w {
			w = l
		}
	}
	if w > nameColMax {
		w = nameColMax
	}
	return w
}
