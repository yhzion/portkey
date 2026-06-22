package cli

import (
	"context"
	"flag"
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
	var checkOnly bool
	var dryRun bool
	var versionTarget string
	var force bool

	return &Command{
		Name:      "update",
		ShortDesc: "Check and install the latest version",
		Flags: func(fs *flag.FlagSet) {
			fs.BoolVar(&checkOnly, "check-only", false,
				"report current vs target version and exit without installing "+
					"(exit "+fmt.Sprintf("%d", ExitUpdateAvailable)+" if an update is available)")
			fs.BoolVar(&dryRun, "dry-run", false,
				"alias for --check-only")
			fs.StringVar(&versionTarget, "version-target", "",
				"install a specific release tag instead of latest (bypasses version comparison)")
			fs.BoolVar(&force, "force", false,
				"reinstall even when already up to date (bypasses version comparison)")
		},
		Run: func(ctx *RunContext) int {
			doCheckOnly := checkOnly || dryRun

			// Check if --version-target was explicitly set (even to "").
			versionTargetSet := false
			ctx.Flags.Visit(func(f *flag.Flag) {
				if f.Name == "version-target" {
					versionTargetSet = true
				}
			})

			// Validate --version-target early, before any network call.
			if versionTargetSet {
				if err := updater.ValidateTag(versionTarget); err != nil {
					fmt.Fprintf(os.Stderr, "Error: --version-target: %v\n", err)
					return ExitRuntime
				}
			}

			// The dev guard fires when the current version is unparseable (e.g.
			// "dev"), UNLESS the user explicitly bypassed version-comparison via
			// --version-target or --force (Task-4 reconciliation). --check-only
			// still needs a parseable current version to produce a meaningful
			// comparison, so the guard fires there too.
			if !versionTargetSet && !force {
				if _, _, _, _, ok := updater.ParseVersion(version); !ok {
					fmt.Printf("Cannot determine current version (%s); skipping self-update.\n", version)
					return ExitSuccess
				}
			}

			if upd == nil {
				upd = updater.DefaultClient()
			}

			// Resolve the target release.
			var rel *updater.Release
			var err error

			if versionTargetSet {
				rel, err = upd.CheckRelease(context.Background(), versionTarget)
			} else {
				rel, err = upd.CheckLatest(context.Background())
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
				return ExitRuntime
			}

			// --check-only / --dry-run: report only, never install.
			if doCheckOnly {
				if updater.IsNewer(version, rel.Tag) {
					fmt.Printf("current: %s, latest: %s\n", version, rel.Tag)
					return ExitUpdateAvailable
				}
				fmt.Printf("up to date (%s)\n", version)
				return ExitSuccess
			}

			// Determine whether to install.
			// --version-target: always install (bypass IsNewer).
			// --force: bypass IsNewer (reinstall even same/older).
			// default: only install when newer.
			switch {
			case versionTargetSet:
				// Explicit tag → install unconditionally.
				fmt.Printf("Installing %s ...\n", rel.Tag)
			case force:
				// Force → reinstall latest.
				fmt.Printf("Reinstalling %s ...\n", rel.Tag)
			default:
				if !updater.IsNewer(version, rel.Tag) {
					fmt.Printf("Already up to date (%s).\n", version)
					return ExitSuccess
				}
				fmt.Printf("Updating %s → %s ...\n", version, rel.Tag)
			}

			progress := func(phase string) {
				fmt.Printf("%s...\n", phase)
			}
			if err := upd.DownloadAndInstall(rel, progress); err != nil {
				fmt.Fprintf(os.Stderr, "Error updating: %v\n", err)
				return ExitRuntime
			}

			fmt.Printf("Updated to %s.\n", rel.Tag)
			return ExitSuccess
		},
	}
}
