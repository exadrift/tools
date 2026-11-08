package templates

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed install-disk.yaml
var installDiskTemplateStr string

//go:embed storage-volume.yaml
var storageVolumeTemplateStr string

//go:embed persistent-volume.yaml
var peristentVolumeTemplateStr string

//go:embed secret.yaml
var secretTemplateStr string

//go:embed namespace.yaml
var namespaceStr string

//go:embed bootstrap-application.yaml
var bootstrapApplicationStr string

//go:embed argocd-application.yaml
var argoCdApplicationStr string

var installDiskTemplate = template.Must(
	template.New("install-disk").Option("missingkey=error").Parse(installDiskTemplateStr),
)

var storageVolumeTemplate = template.Must(
	template.New("storage-volume").Option("missingkey=error").Parse(storageVolumeTemplateStr),
)

var persistentVolumeTemplate = template.Must(
	template.New("persistent-volume").Option("missingkey=error").Parse(peristentVolumeTemplateStr),
)

var secretTemplate = template.Must(
	template.New("secret").Option("missingkey=error").Parse(secretTemplateStr),
)

var namespaceTemplate = template.Must(
	template.New("namespace").Option("missingkey=error").Parse(namespaceStr),
)

var bootstrapApplicationTemplate = template.Must(
	template.New("bootstrap-application").Option("missingkey=error").Parse(bootstrapApplicationStr),
)

var argoCdApplicationStrTemplate = template.Must(
	template.New("argocd-application").Option("missingkey=error").Parse(argoCdApplicationStr),
)

type installDiskTemplateConfig struct {
	Serial string
}

type StorageVolumeConfig struct {
	Name    string
	Serial  string
	MinSize int
	MaxSize int
}

type PersistentVolumeConfig struct {
	Name     string
	Size     int
	Hostname string
}

type SecretConfig struct {
	Name      string
	Namespace string
	Data      map[string]string
}

type NamespaceConfig struct {
	Name string
}

type BootstrapApplicationConfig struct {
	Name string
	Repo string
}

type ArgoCdApplicationConfig struct {
	HelmRepo string
	Repo     string
	Version  string
}

func RenderDiskTemplate(serial string) ([]byte, error) {
	var buf bytes.Buffer

	err := installDiskTemplate.Execute(&buf, installDiskTemplateConfig{
		Serial: serial,
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func RenderStorageVolumeTemplate(storageVolumeConfig StorageVolumeConfig) ([]byte, error) {
	var buf bytes.Buffer

	err := storageVolumeTemplate.Execute(&buf, storageVolumeConfig)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func RenderPersistentVolumeTemplate(pvConfig PersistentVolumeConfig) ([]byte, error) {
	var buf bytes.Buffer

	err := persistentVolumeTemplate.Execute(&buf, pvConfig)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func RenderSecretTemplate(secretConfig SecretConfig) ([]byte, error) {
	var buf bytes.Buffer

	err := secretTemplate.Execute(&buf, secretConfig)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func RenderNamespaceTemplate(namespaceConfig NamespaceConfig) ([]byte, error) {
	var buf bytes.Buffer

	err := namespaceTemplate.Execute(&buf, namespaceConfig)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func RenderBootstrapApplicationTemplate(bootstrapConfig BootstrapApplicationConfig) ([]byte, error) {
	var buf bytes.Buffer

	err := bootstrapApplicationTemplate.Execute(&buf, bootstrapConfig)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func RenderArgoCdApplicationTemplate(argoConfig ArgoCdApplicationConfig) ([]byte, error) {
	var buf bytes.Buffer

	err := argoCdApplicationStrTemplate.Execute(&buf, argoConfig)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
