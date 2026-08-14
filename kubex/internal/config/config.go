package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/exadrift/go/ansi"
)

type KeyBindingsDefintion struct {
	NavNext    string `json:"navNext"`
	NavPrev    string `json:"navPrev"`
	Up         string `json:"up"`
	Down       string `json:"down"`
	ScrollUp   string `json:"scrollUp"`
	ScrollDown string `json:"scrollDown"`
}

type KeyBindings struct {
	NavNext    *ansi.KeyCombo
	NavPrev    *ansi.KeyCombo
	Up         *ansi.KeyCombo
	Down       *ansi.KeyCombo
	ScrollUp   *ansi.KeyCombo
	ScrollDown *ansi.KeyCombo
}

type Config struct {
	KeyBindingsDefintion KeyBindingsDefintion `json:"keyBindings"`
	KeyBindings          *KeyBindings         `json:"-"`
	Location             string               `json:"-"`
}

func defaultField(value string, dflt string) string {
	if value != "" {
		return value
	}

	return dflt
}

// Load loads a new Config file from disk, if none exists, one will be initialized in the
// .config directory of the user's home directory
func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDirPath := filepath.Join(homeDir, ".config")
	_, err = os.Stat(configDirPath)
	if err != nil {
		// path likely doesn't exist, try making it
		err = os.MkdirAll(configDirPath, 0775)
		if err != nil {
			return nil, fmt.Errorf("unable to create directory %s: %w", configDirPath, err)
		}
	}

	var config *Config = &Config{}
	configFilePath := filepath.Join(configDirPath, "kubex.json")
	fileData, err := os.ReadFile(configFilePath)
	if err == nil {
		err = json.Unmarshal(fileData, config)
		if err != nil {
			config = nil
		}
	} else {
		config = nil
	}
	if config == nil {
		config = &Config{}
		fileData = []byte{}
	}

	config.KeyBindingsDefintion.NavNext = defaultField(config.KeyBindingsDefintion.NavNext, "tab")
	config.KeyBindingsDefintion.NavPrev = defaultField(config.KeyBindingsDefintion.NavPrev, "shift+tab")
	config.KeyBindingsDefintion.Up = defaultField(config.KeyBindingsDefintion.Up, "up")
	config.KeyBindingsDefintion.Down = defaultField(config.KeyBindingsDefintion.Down, "down")
	config.KeyBindingsDefintion.ScrollUp = defaultField(config.KeyBindingsDefintion.ScrollUp, "ctrl+pgup")
	config.KeyBindingsDefintion.ScrollDown = defaultField(config.KeyBindingsDefintion.ScrollDown, "ctrl+pgdn")

	cBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, err
	}

	// we do this to ensure if schema changes, updates are persisted
	if !bytes.Equal(cBytes, fileData) {
		if err = os.WriteFile(configFilePath, cBytes, 0775); err != nil {
			return nil, err
		}
	}

	config.KeyBindings = &KeyBindings{}
	if config.KeyBindings.NavNext, err = ansi.ParseHumanName(config.KeyBindingsDefintion.NavNext); err != nil {
		return nil, fmt.Errorf("navNext key binding was invalid: %s", config.KeyBindingsDefintion.NavNext)
	}
	if config.KeyBindings.NavPrev, err = ansi.ParseHumanName(config.KeyBindingsDefintion.NavPrev); err != nil {
		return nil, fmt.Errorf("navPrev key binding was invalid: %s", config.KeyBindingsDefintion.NavPrev)
	}
	if config.KeyBindings.Up, err = ansi.ParseHumanName(config.KeyBindingsDefintion.Up); err != nil {
		return nil, fmt.Errorf("up key binding was invalid: %s", config.KeyBindingsDefintion.Up)
	}
	if config.KeyBindings.Down, err = ansi.ParseHumanName(config.KeyBindingsDefintion.Down); err != nil {
		return nil, fmt.Errorf("down key binding was invalid: %s", config.KeyBindingsDefintion.Down)
	}
	if config.KeyBindings.ScrollUp, err = ansi.ParseHumanName(config.KeyBindingsDefintion.ScrollUp); err != nil {
		return nil, fmt.Errorf("scrollUp key binding was invalid: %s", config.KeyBindingsDefintion.ScrollUp)
	}
	if config.KeyBindings.ScrollDown, err = ansi.ParseHumanName(config.KeyBindingsDefintion.ScrollDown); err != nil {
		return nil, fmt.Errorf("scrollDown key binding was invalid: %s", config.KeyBindingsDefintion.ScrollDown)
	}

	config.Location = configFilePath

	return config, nil
}
