package cli

import (
	"github.com/yhzion/portkey/internal/config"
	"github.com/yhzion/portkey/internal/ssh"
)

// SSHRunner runs an interactive SSH session for a host. Defined here, in the
// consumer package, so the connect command can be tested without a real ssh
// binary ("accept interfaces, return structs").
type SSHRunner interface {
	Run(host config.Host) error
}

// sshRunner is the default SSHRunner, wrapping the ssh package's Run function.
type sshRunner struct{}

func (sshRunner) Run(host config.Host) error { return ssh.Run(host) }

// defaultSSHRunner is used by the connect command. Tests swap it for a fake.
var defaultSSHRunner SSHRunner = sshRunner{}
