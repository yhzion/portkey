package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yhzion/portkey/internal/updater"
)

// updateModel encapsulates the auto-update state and logic.
type updateModel struct {
	tag           string                 // available update tag (non-empty when update available)
	latestRelease *updater.Release       // release info for the available update
	version       string                 // current app version
	checker       UpdateChecker          // update checker (nil in tests/dev)
	checkFailKind updater.CheckErrorKind // set when the last check failed; KindUnknown means no failure
}

// checkUpdate returns a command that checks for a newer release.
func (u *updateModel) checkUpdate() tea.Cmd {
	return func() tea.Msg {
		rel, err := u.checker.CheckLatest()
		if err != nil {
			return updateCheckFailedMsg{Kind: updater.ClassifyCheckError(err)}
		}
		if updater.IsNewer(u.version, rel.Tag) {
			return updateAvailableMsg{Tag: rel.Tag, Rel: rel}
		}
		return nil
	}
}

// checkFailHint returns a short dim hint string for the last failed update
// check, or "" when no failure occurred. Used by the host-list view.
func (u *updateModel) checkFailHint() string {
	switch u.checkFailKind {
	case updater.KindOffline:
		return "(update check: offline)"
	case updater.KindRateLimited:
		return "(update check: GitHub rate-limited — try later)"
	case updater.KindNotFound:
		return "(update check: no releases yet)"
	case updater.KindOther:
		return "(update check failed)"
	default:
		return ""
	}
}

// handleConfirmKey handles y/n/Esc on the update confirmation screen.
// Returns the target screen and an optional command.
func (u *updateModel) handleConfirmKey(msg tea.KeyMsg, keys keyMap) (screen, tea.Cmd) {
	if key.Matches(msg, keys.Escape) {
		return screenHostList, nil
	}
	switch strings.ToLower(msg.String()) {
	case "y":
		return screenHostList, func() tea.Msg {
			return updateDoneMsg{err: nil}
		}
	case "n":
		return screenHostList, nil
	}
	return screenUpdateConfirm, nil
}

// renderConfirm renders the update confirmation prompt.
func (u *updateModel) renderConfirm() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(accentStyle.Render(fmt.Sprintf("✨ New version (%s) detected. Update now?", u.tag)))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("[y] yes / [n] no"))
	return b.String()
}
