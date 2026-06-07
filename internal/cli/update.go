package cli

import (
	"fmt"
	"os"

	"github.com/yhzion/portkey/internal/updater"
)

func runUpdate(args []string, version string, upd *updater.Client) int {
	if hasHelp(args) {
		fmt.Print(helpUpdate())
		return ExitSuccess
	}

	if upd == nil {
		upd = updater.DefaultClient()
	}

	rel, err := upd.CheckLatest()
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
}
