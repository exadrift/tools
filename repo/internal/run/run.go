package run

import (
	"errors"
	"fmt"
	"os/exec"
)

// Run executes a command and on error, will return a new error object, capturing stderr
func Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("error %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}

		return err
	}

	return nil
}

// RunC executes a command and on error, will return a new error object, capturing stderr
// Captures the stdout
func RunC(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	oBytes, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("error %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}

		return "", err
	}

	return string(oBytes), nil
}

func ShellRgb(red int, green int, blue int, text string) string {
	return fmt.Sprintf(`\[\e[38;2;%d;%d;%dm\]%s\[\e[0m\]`, red, green, blue, text)
}
