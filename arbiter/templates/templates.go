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

var installDiskTemplate = template.Must(
	template.New("install-disk").Option("missingkey=error").Parse(installDiskTemplateStr),
)

var storageVolumeTemplate = template.Must(
	template.New("storage-volume").Option("missingkey=error").Parse(storageVolumeTemplateStr),
)

var persistentVolumeTemplate = template.Must(
	template.New("persistent-volume").Option("missingkey=error").Parse(peristentVolumeTemplateStr),
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
