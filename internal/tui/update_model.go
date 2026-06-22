package tui

import (
	"context"
	"errors"
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
	installer     Installer              // installs updates; raw client, NOT the caching checker (nil in tests/dev)
	checkFailKind updater.CheckErrorKind // set when the last check failed; KindUnknown means no failure

	// cancelCheck cancels an in-flight checkUpdate Cmd. It is set when
	// checkUpdate is called and invoked when the app quits so a hung check
	// does not linger in the background.
	cancelCheck context.CancelFunc
}

// checkUpdate returns a command that checks for a newer release.
// The provided ctx is used as the parent; checkUpdate derives a cancellable
// child context and stores the cancel func in u.cancelCheck so callers can
// abort the check (e.g. on quit). Passing context.Background() is fine for
// the normal Init path; tests pass their own ctx.
func (u *updateModel) checkUpdate(ctx context.Context) tea.Cmd {
	ctx, cancel := context.WithCancel(ctx)
	u.cancelCheck = cancel
	return func() tea.Msg {
		rel, err := u.checker.CheckLatest(ctx)
		if err != nil {
			// A context-cancelled check means the user quit — surface nothing.
			if errors.Is(err, context.Canceled) {
				return nil
			}
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
		inst := u.installer
		rel := u.latestRelease
		return screenHostList, func() tea.Msg {
			if inst == nil || rel == nil {
				return updateDoneMsg{err: errors.New("no installer configured")}
			}
			return updateDoneMsg{err: inst.DownloadAndInstall(rel, nil)}
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
