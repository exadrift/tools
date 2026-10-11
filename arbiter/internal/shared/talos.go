package shared

import (
	"fmt"
	"os"
	"strings"
)

const (
	NameRefFile string = ".nameref"
)

func GetNameFromRef() (string, error) {
	fBytes, err := os.ReadFile(NameRefFile)
	if err != nil {
		return "", fmt.Errorf("unable to load .nameref file, are you sure you're in the right directory?: %w", err)
	}

	s := string(fBytes)
	name := strings.Trim(s, "\n \t")
	if len(name) == 0 {
		return "", fmt.Errorf("name ref file did not contain a name")
	}

	return name, nil
}
