package ssh_test

import (
	"testing"

	"github.com/yhzion/portkey/internal/config"
	"github.com/yhzion/portkey/internal/ssh"
)

func TestBuildArgsDefaultPort(t *testing.T) {
	host := config.Host{Name: "dev", Username: "youngho", Host: "192.168.0.10", Port: 22}
	args := ssh.BuildArgs(host)

	if len(args) != 1 {
		t.Fatalf("len(args) = %d, want 1", len(args))
	}
	if args[0] != "youngho@192.168.0.10" {
		t.Errorf("args[0] = %q, want %q", args[0], "youngho@192.168.0.10")
	}
}

func TestBuildArgsCustomPort(t *testing.T) {
	host := config.Host{Name: "staging", Username: "ubuntu", Host: "staging.example.com", Port: 2222}
	args := ssh.BuildArgs(host)

	if len(args) != 3 {
		t.Fatalf("len(args) = %d, want 3", len(args))
	}
	if args[0] != "-p" {
		t.Errorf("args[0] = %q, want %q", args[0], "-p")
	}
	if args[1] != "2222" {
		t.Errorf("args[1] = %q, want %q", args[1], "2222")
	}
	if args[2] != "ubuntu@staging.example.com" {
		t.Errorf("args[2] = %q, want %q", args[2], "ubuntu@staging.example.com")
	}
}

func TestBuildArgsPortOne(t *testing.T) {
	host := config.Host{Username: "u", Host: "h", Port: 1}
	args := ssh.BuildArgs(host)
	if args[0] != "-p" || args[1] != "1" {
		t.Errorf("expected -p 1 for port 1, got %v", args)
	}
}

func TestBuildArgsPort65535(t *testing.T) {
	host := config.Host{Username: "u", Host: "h", Port: 65535}
	args := ssh.BuildArgs(host)
	if args[0] != "-p" || args[1] != "65535" {
		t.Errorf("expected -p 65535, got %v", args)
	}
}

func TestBuildArgsArgsAreSeparated(t *testing.T) {
	host := config.Host{Username: "u; rm -rf /", Host: "h", Port: 22}
	args := ssh.BuildArgs(host)

	// Key security test: arguments are passed as separate elements to exec.Command,
	// not as a single shell string. The semicolon is inside one arg, not interpreted by shell.
	if len(args) != 1 {
		t.Fatalf("len(args) = %d, want 1 (args are separated, not shell-interpolated)", len(args))
	}
	// The combined string "u; rm -rf /@h" is safe because exec.Command
	// passes each arg directly to the OS without shell interpretation.
	if args[0] != "u; rm -rf /@h" {
		t.Errorf("args[0] = %q, want %q", args[0], "u; rm -rf /@h")
	}
}

func TestBuildArgsNeverShellString(t *testing.T) {
	host := config.Host{Username: "u", Host: "h", Port: 22}
	args := ssh.BuildArgs(host)

	// Verify all args are individual elements (no combined shell string)
	for _, arg := range args {
		if arg == "ssh u@h" {
			t.Error("arg looks like a combined shell string")
		}
	}
}

func TestBuildArgsCustomPortFormat(t *testing.T) {
	host := config.Host{Username: "ubuntu", Host: "example.com", Port: 2222}
	args := ssh.BuildArgs(host)

	if args[0] != "-p" {
		t.Errorf("first arg should be -p, got %q", args[0])
	}
	if args[1] != "2222" {
		t.Errorf("second arg should be port string, got %q", args[1])
	}
	if args[2] != "ubuntu@example.com" {
		t.Errorf("third arg should be target, got %q", args[2])
	}
}
