package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/exadrift/tools/arbiter/internal/crypto"
	yaml "github.com/goccy/go-yaml"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	machineryconfig "github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/configloader"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
)

const (
	machineSecretsFilename     = "machine-secrets.encrypted"
	controlPlaneSecretFilename = "controlplane.encrypted"
	talosConfigSecretFilename  = "talosconfig.encrypted"
	renderedConfigsSubpath     = "rendered"
	nameReferenceFilename      = ".nameref"

	InstallDiskFilename = "config/install-disk.yaml"
)

type Config struct {
	configDir  string         `yaml:"-"`
	Crypto     *crypto.Crypto `yaml:"-"`
	Key        string         `yaml:"privateKey"`
	ProjectDir string         `yaml:"projectDir"`
	IpAddress  string         `yaml:"ipAddress"`
	Name       string         `yaml:"-"`
}

type MachineConfig struct {
	configDir        string
	IpAddress        string
	SecretBundle     *secrets.Bundle
	ControlPlane     machineryconfig.Provider
	TalosConfig      *clientconfig.Config
	BootDiskSelector []byte
}

func (mc *MachineConfig) RenderControlPlane() ([]byte, error) {
	var controlPlane []byte

	controlPlaneYaml, err := mc.ControlPlane.Bytes()
	if err != nil {
		return nil, fmt.Errorf("unable to serialize control plane yaml: %w", err)
	}

	controlPlane = append(controlPlane, controlPlaneYaml...)
	controlPlane = append(controlPlane, []byte("\n---\n")...)
	controlPlane = append(controlPlane, bytes.TrimSpace(mc.BootDiskSelector)...)
	controlPlane = append(controlPlane, '\n')

	if _, err = configloader.NewFromBytes(controlPlane); err != nil {
		return nil, fmt.Errorf("machine config contained errors: %w", err)
	}

	return controlPlane, nil
}

func (mc *MachineConfig) CleanUp() {
	_ = os.RemoveAll(filepath.Join(mc.configDir, renderedConfigsSubpath))
}

func New(name string, projectDir string, ipAddress string, crypt *crypto.Crypto) (*Config, error) {
	if projectDir == "" || ipAddress == "" || name == "" || crypt == nil {
		return nil, fmt.Errorf("missing a required argumnt")
	}

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

	if err = os.MkdirAll(configPath, 0700); err != nil {
		return err
	}

	if err = os.WriteFile(filepath.Join(configPath, "config.yaml"), cBytes, 0600); err != nil {
		return fmt.Errorf("unable to write config.yaml: %w", err)
	}

	return nil
}

func (c *Config) secretsPath() string {
	return filepath.Join(c.ProjectDir, "secrets")
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

	return LoadFromAltLocation(filepath.Join(configPath, "config.yaml"), name)
}

func LoadFromAltLocation(configPath string, name string) (*Config, error) {
	cBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("unable to load config.yaml: %w", err)
	}

	cfg := &Config{}
	if err = yaml.Unmarshal(cBytes, cfg); err != nil {
		return nil, fmt.Errorf("unable to deserialize config.yaml: %w", err)
	}

	cfg.Name = name
	cfg.configDir = configPath

	c, err := crypto.Deserialize(cfg.Key)
	if err != nil {
		return nil, fmt.Errorf("unable to deserilize private key: %w", err)
	}
	cfg.Crypto = c

	return cfg, nil
}

func (c *Config) GetConfigDir() string {
	return c.configDir
}

func (c *Config) StoreData(relPath string, data []byte) error {
	filename := filepath.Join(c.ProjectDir, relPath)

	// check to see if the base directory exists
	baseDir, _ := filepath.Split(filename)
	if _, err := os.Stat(baseDir); err != nil {
		if err = os.MkdirAll(baseDir, 0755); err != nil {
			return err
		}
	}

	return os.WriteFile(filename, data, 0644)
}

func (c *Config) LoadData(relPath string) ([]byte, error) {
	filename := filepath.Join(c.ProjectDir, relPath)
	return os.ReadFile(filename)
}

func (c *Config) EncryptAndStoreData(filename string, data []byte) error {
	// check to see if the base directory exists
	baseDir, _ := filepath.Split(filename)
	if _, err := os.Stat(baseDir); err != nil {
		if err = os.MkdirAll(baseDir, 0755); err != nil {
			return err
		}
	}

	encString, err := c.Crypto.Encrypt(data)
	if err != nil {
		return fmt.Errorf("unable to encrypt data: %w", err)
	}

	return os.WriteFile(filename, []byte(encString), 0644)
}

func (c *Config) DecryptStoredData(filename string) ([]byte, error) {
	encBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to load encrypted file: %w", err)
	}

	dec, err := c.Crypto.Decrypt(encBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt file: %w", err)
	}

	return []byte(dec), nil
}

func (c *Config) LoadMachineSecrets() (*secrets.Bundle, error) {
	secretsFile := filepath.Join(c.secretsPath(), machineSecretsFilename)

	b, err := c.DecryptStoredData(secretsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve machine secrets: %w", err)
	}

	bundle := &secrets.Bundle{
		Clock: secrets.NewClock(),
	}
	decBytes := []byte(b)
	if err = json.Unmarshal(decBytes, bundle); err != nil {
		return nil, fmt.Errorf("unable to unmarshal secrets bundle: %w", err)
	}

	return bundle, nil
}

func (c *Config) SaveMachineSecrets(secretsBundle *secrets.Bundle) error {
	sBytes, err := json.Marshal(secretsBundle)
	if err != nil {
		return fmt.Errorf("unable to marshal secrets bundle to json: %w", err)
	}

	return c.EncryptAndStoreData(filepath.Join(c.secretsPath(), machineSecretsFilename), sBytes)
}

func (c *Config) SaveControlPlaneConfig(provider machineryconfig.Provider) error {
	pBytes, err := provider.Bytes()
	if err != nil {
		return fmt.Errorf("unable to marshal control plane config to json: %w", err)
	}

	return c.EncryptAndStoreData(filepath.Join(c.secretsPath(), controlPlaneSecretFilename), pBytes)
}

func (c *Config) LoadControlPlaneConfig() (machineryconfig.Provider, error) {
	pBytes, err := c.DecryptStoredData(filepath.Join(c.secretsPath(), controlPlaneSecretFilename))
	if err != nil {
		return nil, fmt.Errorf("unable to load control plane encrypted configs: %w", err)
	}

	cfg, err := configloader.NewFromBytes(pBytes)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) SaveTalosConfig(talosConfig *clientconfig.Config) error {
	pBytes, err := talosConfig.Bytes()
	if err != nil {
		return fmt.Errorf("unable to marshal talosconfig to json: %w", err)
	}

	return c.EncryptAndStoreData(filepath.Join(c.secretsPath(), talosConfigSecretFilename), pBytes)
}

func (c *Config) LoadTalosConfig() (*clientconfig.Config, error) {
	pBytes, err := c.DecryptStoredData(filepath.Join(c.secretsPath(), talosConfigSecretFilename))
	if err != nil {
		return nil, fmt.Errorf("unable to load talosconfig encrypted configs: %w", err)
	}

	return clientconfig.FromBytes(pBytes)
}

func (c *Config) LoadMachineConfig() (*MachineConfig, error) {
	secrets, err := c.LoadMachineSecrets()
	if err != nil {
		return nil, err
	}

	controlPlane, err := c.LoadControlPlaneConfig()
	if err != nil {
		return nil, err
	}

	bootDiskSelector, err := c.LoadData(InstallDiskFilename)
	if err != nil {
		return nil, err
	}

	talosConfig, err := c.LoadTalosConfig()
	if err != nil {
		return nil, err
	}

	return &MachineConfig{
		configDir:        c.configDir,
		IpAddress:        c.IpAddress,
		SecretBundle:     secrets,
		ControlPlane:     controlPlane,
		BootDiskSelector: bootDiskSelector,
		TalosConfig:      talosConfig,
	}, nil
}

func GetNameRef(projectDir string) (string, error) {
	rBytes, err := os.ReadFile(filepath.Join(projectDir, nameReferenceFilename))
	if err != nil {
		return "", err
	}

	return strings.Trim(string(rBytes), " \t\n"), nil
}

func (c *Config) SaveNameRef() error {
	if err := os.WriteFile(filepath.Join(c.ProjectDir, nameReferenceFilename), []byte(c.Name), 0644); err != nil {
		return fmt.Errorf("unable to update name reference file: %w", err)
	}
	return nil
}
