package kubectl

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func run(name string, args ...string) ([]string, error) {
	cmd := exec.Command(name, args...)
	b, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("terminated with exit code %d - %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}

		return nil, err
	}

	output := string(b)
	lines := strings.Split(output, "\n")
	var cleansed []string
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		cleansed = append(cleansed, line)
	}

	return cleansed, nil
}

func GetContexts() ([]string, error) {
	return run("kubectl", "config", "get-contexts", "--output", "name")
}

func GetCurrentContext() (string, error) {
	all, err := run("kubectl", "config", "current-context")
	if err != nil {
		return "", err
	}

	if len(all) == 0 {
		return "", fmt.Errorf("no context available")
	}

	return all[0], nil
}

func SetCurrentContext(ctx string) error {
	_, err := run("kubectl", "config", "use-context", ctx)
	return err
}

func GetNamespaces() ([]string, error) {
	nsList, err := run("kubectl", "get", "ns", "--output", "name")
	if err != nil {
		return nil, err
	}

	strippedList := make([]string, len(nsList))
	for i, ns := range nsList {
		strippedList[i] = strings.TrimPrefix(ns, "namespace/")
	}

	return strippedList, nil
}

func GetCurrentNamespace() (string, error) {
	all, err := run("kubectl", "config", "view", "--output", "jsonpath={.contexts[0].context.namespace}")
	if err != nil {
		return "", err
	}

	if len(all) == 0 {
		return "default", nil
	}

	return all[0], nil
}

func SetCurrentNamespace(ns string) error {
	_, err := run("kubectl", "config", "set-context", "--current", "--namespace", ns)
	return err
}
