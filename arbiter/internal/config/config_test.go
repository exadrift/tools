package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractIndex(t *testing.T) {
	i, err := extractIndexFromFilename("0001-hello_world.yaml")
	assert.NoError(t, err)
	assert.Equal(t, 1, i)
}
