package tailnet

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Puller copies a peer's remote ~/.claude/projects into a local mirror dir.
type Puller interface {
	// Pull mirrors remoteDir (relative to the remote home) on host into
	// mirrorDir on the local machine.
	Pull(host, remoteDir, mirrorDir string) error
}

// SSHPuller pulls over `tailscale ssh <host>` (no user@ — Tailscale ACLs
// resolve the login user). It prefers rsync for incremental transfer and falls
// back to a tar stream if rsync is unavailable or fails.
type SSHPuller struct {
	timeout time.Duration
}

func NewSSHPuller(timeout time.Duration) *SSHPuller {
	return &SSHPuller{timeout: timeout}
}

func (p *SSHPuller) Pull(host, remoteDir, mirrorDir string) error {
	if err := os.MkdirAll(mirrorDir, 0755); err != nil {
		return fmt.Errorf("creating mirror dir: %w", err)
	}

	rsyncErr := p.rsync(host, remoteDir, mirrorDir)
	if rsyncErr == nil {
		return nil
	}
	if tarErr := p.tarStream(host, remoteDir, mirrorDir); tarErr != nil {
		return fmt.Errorf("rsync failed (%v); tar fallback failed: %w", rsyncErr, tarErr)
	}
	return nil
}

// rsync runs: rsync -az --timeout=N -e 'tailscale ssh' <host>:<remoteDir>/ <mirrorDir>/
// Trailing slashes copy the *contents* of the remote dir into the mirror.
func (p *SSHPuller) rsync(host, remoteDir, mirrorDir string) error {
	if _, err := exec.LookPath("rsync"); err != nil {
		return fmt.Errorf("rsync not found: %w", err)
	}
	secs := int(p.timeout.Seconds())
	if secs < 1 {
		secs = 20
	}
	args := []string{
		"-az",
		"--timeout=" + strconv.Itoa(secs),
		"-e", "tailscale ssh",
		host + ":" + ensureSlash(remoteDir),
		ensureSlash(mirrorDir),
	}
	cmd := exec.Command("rsync", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rsync: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// tarStream runs: tailscale ssh <host> -- sh -c 'tar -C "$HOME" -cf - <remoteDir>'
// piped into `tar -x` under mirrorDir's parent. The remote shell resolves $HOME
// so the hub never needs the remote home path.
func (p *SSHPuller) tarStream(host, remoteDir, mirrorDir string) error {
	// Mirror layout is <root>/<host>/projects; we extract <remoteDir> (e.g.
	// ".claude/projects") and relocate its leaf into mirrorDir.
	extractRoot, err := os.MkdirTemp("", "cctrack-pull-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(extractRoot)

	remoteCmd := fmt.Sprintf("tar -C \"$HOME\" -cf - %s", shellQuote(remoteDir))
	send := exec.Command("tailscale", "ssh", host, "--", "sh", "-c", remoteCmd)
	recv := exec.Command("tar", "-C", extractRoot, "-xf", "-")

	pipe, err := send.StdoutPipe()
	if err != nil {
		return err
	}
	recv.Stdin = pipe

	if err := recv.Start(); err != nil {
		return err
	}
	if err := send.Run(); err != nil {
		recv.Wait()
		return fmt.Errorf("tailscale ssh tar: %w", err)
	}
	if err := recv.Wait(); err != nil {
		return fmt.Errorf("local tar extract: %w", err)
	}

	// Move the extracted leaf (<extractRoot>/<remoteDir>) into mirrorDir.
	extracted := joinClean(extractRoot, remoteDir)
	return mergeTree(extracted, mirrorDir)
}

func ensureSlash(p string) string {
	if strings.HasSuffix(p, "/") {
		return p
	}
	return p + "/"
}
