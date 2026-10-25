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

var installDiskTemplate = template.Must(
	template.New("install-disk").Option("missingkey=error").Parse(installDiskTemplateStr),
)

var storageVolumeTemplate = template.Must(
	template.New("storage-volume").Option("missingkey=error").Parse(storageVolumeTemplateStr),
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
