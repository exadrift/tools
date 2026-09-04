package manifest

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type Package struct {
	// Name of package to sync
	Name string `yaml:"name"`

	// Version of package to sync
	Version string `yaml:"version"`

	// Command which returns the installed package version
	GetVersion string `yaml:"getVersion"`

	// Inline command/command set to execute in order to sync
	Sync string `yaml:"sync"`

	// Script to execute with sync instructions
	SyncScript string `yaml:"syncScript"`
}

type Manifest struct {
	// Version of manifest
	Version string `yaml:"version"`

	// Packages will be synced in order of occurrence
	Packages []Package `yaml:"packages"`

	// the default shell to use (or full path)
	Shell string `yaml:"shell"`
}

func FromFile(filename string) (*Manifest, error) {
	manifestData, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	manifest := &Manifest{}
	if err = yaml.Unmarshal(manifestData, manifest); err != nil {
		return nil, err
	}

	if len(manifest.Packages) == 0 {
		return nil, fmt.Errorf("manifest contains no packages")
	}

	for _, pkg := range manifest.Packages {
		if pkg.Name == "" {
			return nil, fmt.Errorf("every package must specify a name")
		}

		if pkg.Version == "" {
			return nil, fmt.Errorf("package %s does not specify a version", pkg.Name)
		}

		if pkg.GetVersion == "" {
			return nil, fmt.Errorf("package %s does not specify a command to get the installed version", pkg.Name)
		}

		if pkg.Sync != "" && pkg.SyncScript != "" {
			return nil, fmt.Errorf("package %s specifies both sync and syncScript, only one may be specified", pkg.Name)
		}
	}

	return manifest, nil
}
