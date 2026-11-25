package config

import (
	"sync"

	cfg "github.com/exadrift/go/config"
)

type appConfig struct {
	OPENAI_ENDPOINT      string `env:"BUDDY_OPENAI_ENDPOINT" default:"localhost:8081"`
	OPENAI_API_KEY       string `env:"BUDDY_OPENAI_API_KEY"`
	OPENAI_MODEL         string `env:"BUDDY_OPENAI_MODEL"`
	WHISPER_ENDPOINT     string `env:"BUDDY_WHISPER_ENDPOINT" default:"localhost:8082"`
	KOKORO_ENDPOINT      string `env:"BUDDY_KOKORO_ENDPOINT" default:"localhost:8083"`
	KOKORO_DEFAULT_VOICE string `env:"BUDDY_KOKORO_DEFAULT_VOICE" default:"jm_kumo"`
	DEFAULT_PROMPT       string `env:"BUDDY_DEFAULT_PROMPT" default:""`
}

var (
	c    *appConfig
	lock sync.Mutex
)

func Get() *appConfig {
	if c == nil {
		lock.Lock()
		defer lock.Unlock()

		ac := &appConfig{}
		if err := cfg.Apply(ac); err != nil {
			panic(err)
		}

		c = ac
	}

	return c
}
