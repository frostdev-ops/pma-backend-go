package git

import (
	"os/exec"
	"strings"
)

// GetCurrentCommitMessage returns the current commit message of the git repository
func GetCurrentCommitMessage() (string, error) {
	cmd := exec.Command("git", "log", "-1", "--pretty=format:%s")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// Get Repo URL
func GetRepoURL() (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// Get Version
func GetVersion() (string, error) {
	cmd := exec.Command("git", "log", "-1", "--pretty=format:%s")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
