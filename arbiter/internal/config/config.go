package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/exadrift/tools/arbiter/internal/crypto"
	yaml "github.com/goccy/go-yaml"
)

type Config struct {
	Crypto     *crypto.Crypto `yaml:"-"`
	Key        string         `yaml:"privateKey"`
	ProjectDir string         `yaml:"projectDir"`
	IpAddress  string         `yaml:"ipAddress"`
	Name       string         `yaml:"-"`
}

func New(name string, projectDir string, ipAddress string, crypt *crypto.Crypto) (*Config, error) {
	cr := &Config{
		Name:       name,
		IpAddress:  ipAddress,
		ProjectDir: projectDir,
		Crypto:     crypt,
	}

	ser, err := crypt.Serialize()
	if err != nil {
		return nil, err
	}
	cr.Key = ser

	return cr, nil
}

func (c *Config) Save() error {
	cBytes, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("unable to save config.yaml: %w", err)
	}

	configPath, err := GetConfigDir(c.Name)
	if err != nil {
		return err
	}

	if err = os.WriteFile(filepath.Join(configPath, "config.yaml"), cBytes, 0600); err != nil {
		return fmt.Errorf("unable to write config.yaml: %w", err)
	}

	return nil
}

func GetConfigDir(projectName string) (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to retrieve home directory: %w", err)
	}

	return filepath.Join(homedir, ".arbiter", projectName), nil
}

func Load(name string) (*Config, error) {
	configPath, err := GetConfigDir(name)
	if err != nil {
		return nil, err
	}

	cBytes, err := os.ReadFile(filepath.Join(configPath, "config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("unable to load config.yaml: %w", err)
	}

	cfg := &Config{}
	if err = yaml.Unmarshal(cBytes, cfg); err != nil {
		return nil, fmt.Errorf("unable to deserialize config.yaml: %w", err)
	}

	cfg.Name = name

	c, err := crypto.Deserialize(cfg.Key)
	if err != nil {
		return nil, fmt.Errorf("unable to deserilize private key: %w", err)
	}
	cfg.Crypto = c

	return cfg, nil
}
