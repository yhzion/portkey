package tui

import "github.com/mattn/go-runewidth"

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
// fall in the cut-off region. Widths are measured in terminal display columns
// (full-width runes count as 2), matching lipgloss's Width, so a column sized
// by the same measure never overflows. Truncation is rune-aware so multi-byte
// names are never split mid-rune.
func truncateName(name string, width int) (string, int) {
	if width <= 0 {
		return "", 0
	}
	runes := []rune(name)
	// A name that fits the column is kept whole; only an over-width name is
	// trimmed to width-1 display columns (each full-width rune = 2) with the
	// last column reserved for the ellipsis.
	if runewidth.StringWidth(name) <= width {
		return name, len(runes)
	}
	kept, contentWidth := 0, 0
	for _, r := range runes {
		rw := runewidth.RuneWidth(r)
		if rw > width-1-contentWidth {
			break
		}
		contentWidth += rw
		kept++
	}
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
// names, clamped to [nameColMin, nameColMax]. Width is measured in display
// columns (full-width runes = 2) to match lipgloss's Width, so the column
// stays tight while the cap bounds a single pathological name.
func nameColumnWidth(names []string) int {
	w := nameColMin
	for _, n := range names {
		if l := runewidth.StringWidth(n); l > w {
			w = l
		}
	}
	if w > nameColMax {
		w = nameColMax
	}
	return w
}
