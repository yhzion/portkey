package tui

import (
	"strings"
	"testing"

	"github.com/yhzion/portkey/internal/config"
)

// A name long enough to wrap under the old fixed Width(20) must now render on a
// single line (the reported readability bug).
func TestRenderHostItem_LongName_DoesNotWrap(t *testing.T) {
	m := newTestModel()
	h := config.Host{Name: "datamaker-192-168-14-135", Username: "datamaker", Host: "192.168.14.135", Port: 22}
	w := nameColumnWidth([]string{h.Name})
	row := m.renderHostItem(0, h, false, nil, w)

	if body := strings.TrimSuffix(row, "\n"); strings.Contains(body, "\n") {
		t.Errorf("row wrapped to multiple lines:\n%q", row)
	}
	// At width == its own length, the name is shown in full (no ellipsis).
	if strings.Contains(row, "…") {
		t.Errorf("name within column width should not be truncated:\n%q", row)
	}
}

// A name past the hard cap is truncated with an ellipsis, still on one line.
func TestRenderHostItem_OverCap_TruncatedSingleLine(t *testing.T) {
	m := newTestModel()
	h := config.Host{Name: "this-is-an-extremely-long-hostname-way-over-the-cap", Username: "u", Host: "h", Port: 22}
	w := nameColumnWidth([]string{h.Name})
	row := m.renderHostItem(0, h, false, nil, w)

	if !strings.Contains(row, "…") {
		t.Errorf("expected ellipsis for over-cap name, got:\n%q", row)
	}
	if body := strings.TrimSuffix(row, "\n"); strings.Contains(body, "\n") {
		t.Errorf("row wrapped to multiple lines:\n%q", row)
	}
}

func TestTruncateName(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		width    int
		wantText string
		wantKept int
	}{
		{"shorter than width is unchanged", "rtx5090", 20, "rtx5090", 7},
		{"equal to width is unchanged", "datamaker-10-0-250-0", 20, "datamaker-10-0-250-0", 20},
		{"longer than width gets ellipsis", "datamaker-192-168-14-135", 20, "datamaker-192-168-1…", 19},
		{"width of one is just ellipsis", "anything", 1, "…", 0},
		{"rune-aware truncation", "한국어서버이름테스트", 5, "한국어서…", 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotText, gotKept := truncateName(c.input, c.width)
			if gotText != c.wantText {
				t.Errorf("truncateName(%q, %d) text = %q, want %q", c.input, c.width, gotText, c.wantText)
			}
			if gotKept != c.wantKept {
				t.Errorf("truncateName(%q, %d) kept = %d, want %d", c.input, c.width, gotKept, c.wantKept)
			}
			// The visible rune count must never exceed width.
			if n := len([]rune(gotText)); n > c.width {
				t.Errorf("truncateName(%q, %d) produced %d runes, exceeds width", c.input, c.width, n)
			}
		})
	}
}

func TestKeepPositions(t *testing.T) {
	// Positions in the truncated-away region (index >= kept) are dropped.
	got := keepPositions([]int{0, 3, 5, 8}, 5)
	want := []int{0, 3}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("keepPositions = %v, want %v", got, want)
	}
	// Input must not be mutated.
	src := []int{0, 9}
	_ = keepPositions(src, 5)
	if src[1] != 9 {
		t.Errorf("keepPositions mutated its input: %v", src)
	}
}

func TestNameColumnWidth(t *testing.T) {
	cases := []struct {
		name  string
		names []string
		want  int
	}{
		{"empty falls back to minimum", nil, nameColMin},
		{"all short clamps to minimum", []string{"a", "bb", "ccc"}, nameColMin},
		{"longest within range wins", []string{"short", "datamaker-192-168-14-135"}, 24},
		{"longest beyond max is capped", []string{"this-is-an-extremely-long-hostname-way-over-the-cap"}, nameColMax},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := nameColumnWidth(c.names); got != c.want {
				t.Errorf("nameColumnWidth(%v) = %d, want %d", c.names, got, c.want)
			}
		})
	}
}
