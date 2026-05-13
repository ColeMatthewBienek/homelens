// Package share uploads an HTML report as a GitHub Gist via the gh CLI.
package share

import (
	"fmt"
	"os/exec"
	"strings"
)

// Gist publishes a single file as a gist and returns its URL.
// Requires `gh` to be installed and authenticated.
func Gist(path string, public bool) (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("`gh` CLI not found on PATH. Install from https://cli.github.com")
	}
	args := []string{"gist", "create"}
	if public {
		args = append(args, "--public")
	}
	args = append(args, path)
	out, err := exec.Command("gh", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh gist create failed: %s", strings.TrimSpace(string(out)))
	}
	// gh prints the gist URL as the last line
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("gh produced no output")
	}
	return strings.TrimSpace(lines[len(lines)-1]), nil
}
