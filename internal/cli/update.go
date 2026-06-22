package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/yhzion/portkey/internal/updater"
)

// updateCmd is a placeholder used for root help display and command lookup.
// Dispatch replaces it with a fresh command from newUpdateCmd at runtime.
var updateCmd = &Command{
	Name:      "update",
	ShortDesc: "Check and install the latest version",
}

func newUpdateCmd(upd *updater.Client, version string) *Command {
	return &Command{
		Name:      "update",
		ShortDesc: "Check and install the latest version",
		Run: func(ctx *RunContext) int {
			if _, _, _, _, ok := updater.ParseVersion(version); !ok {
				fmt.Printf("Cannot determine current version (%s); skipping self-update.\n", version)
				return ExitSuccess
			}

			if upd == nil {
				upd = updater.DefaultClient()
			}

			rel, err := upd.CheckLatest(context.Background())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
				return ExitRuntime
			}

			if !updater.IsNewer(version, rel.Tag) {
				fmt.Printf("Already up to date (%s).\n", version)
				return ExitSuccess
			}

			fmt.Printf("Updating %s → %s ...\n", version, rel.Tag)

			if err := upd.DownloadAndInstall(rel); err != nil {
				fmt.Fprintf(os.Stderr, "Error updating: %v\n", err)
				return ExitRuntime
			}

			fmt.Printf("Updated to %s.\n", rel.Tag)
			return ExitSuccess
		},
	}
}
