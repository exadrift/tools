package kubectl

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Context struct {
	Namespace string `json:"namespace"`
}

type NamedContext struct {
	Name    string  `json:"name"`
	Context Context `json:"context"`
}

type Config struct {
	Contexts []NamedContext `json:"contexts"`
}

func run_pre_split(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	b, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("terminated with exit code %d - %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}

		return nil, err
	}

	return b, nil
}

func run(name string, args ...string) ([]string, error) {
	b, err := run_pre_split(name, args...)
	if err != nil {
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

func GetCurrentNamespace(curContext string) (string, error) {
	jsonBytes, err := run_pre_split("kubectl", "config", "view", "--output", "json")
	if err != nil {
		return "", err
	}

	cfg := Config{}
	if err = json.Unmarshal(jsonBytes, &cfg); err != nil {
		return "", fmt.Errorf("unable to unmarshal config data: %w", err)
	}

	for _, ctx := range cfg.Contexts {
		if ctx.Name == curContext {
			return ctx.Context.Namespace, nil
		}
	}

	return "default", nil
}

func SetCurrentNamespace(ns string) error {
	_, err := run("kubectl", "config", "set-context", "--current", "--namespace", ns)
	return err
}
