package templates

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed install-disk.yaml
var installDiskTemplateStr string

var installDiskTemplate = template.Must(
	template.New("install-disk").Option("missingkey=error").Parse(installDiskTemplateStr),
)

type installDiskTemplateConfig struct {
	Serial string
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
