package ssh

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/yhzion/portkey/internal/config"
)

// BuildArgs converts a Host into ssh command arguments.
func BuildArgs(host config.Host) []string {
	args := []string{}

	if host.Port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", host.Port))
	}

	target := fmt.Sprintf("%s@%s", host.Username, host.Host)
	args = append(args, target)
	return args
}

// DefaultRunner executes ssh commands via exec.Command.
type DefaultRunner struct{}

// Run executes a command with the given arguments and I/O streams.
func (r *DefaultRunner) Run(name string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// Run executes ssh for the given host config.
func Run(host config.Host) error {
	args := BuildArgs(host)
	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
