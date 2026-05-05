package handlers

import (
	"encoding/json"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Config struct {
	Enable bool `json:"enable"`
}

var (
	currentConfig = Config{Enable: true}
	configMu      sync.RWMutex
)

func GetConfig() Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return currentConfig
}

func setConfig(c Config) {
	configMu.Lock()
	defer configMu.Unlock()
	currentConfig = c
}

func RequestConfig(context *glsp.Context) error {
	section := "goTmplSupport"
	params := protocol.ConfigurationParams{
		Items: []protocol.ConfigurationItem{
			{Section: &section},
		},
	}

	log.Debug().Any("params", params).Msg("requesting config from client")

	var result []json.RawMessage
	context.Call("workspace/configuration", params, &result)

	if len(result) > 0 {
		var c Config
		if err := json.Unmarshal(result[0], &c); err != nil {
			log.Error().Err(err).Msg("failed to parse config")
		} else {
			setConfig(c)
			log.Debug().Any("config", c).Msg("config stored")
		}
	}
	return nil
}

func ConfigChanged(context *glsp.Context, _ *protocol.DidChangeConfigurationParams) error {
	go RequestConfig(context)

	return nil
}

func SetTrace(_ *glsp.Context, params *protocol.SetTraceParams) error {
	log.Debug().Any("params", params).Msg("SetTrace")
	protocol.SetTraceValue(params.Value)

	switch params.Value {
	case protocol.TraceValueOff:
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case protocol.TraceValueMessage:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case protocol.TraceValueVerbose:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Debug().Any("trace value", params.Value).Msg("trace level set")

	return nil
}
