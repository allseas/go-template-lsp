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
	EnableServer bool `json:"enableServer"`
	Trace        struct {
		Server protocol.TraceValue `json:"server"`
	} `json:"trace"`
}

var (
	currentConfig = Config{EnableServer: true}
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

func traceFromConfig(c Config) protocol.TraceValue {
	if c.Trace.Server != "" {
		return c.Trace.Server
	}

	return protocol.TraceValueMessage
}

func applyTraceLevel(trace protocol.TraceValue) {
	protocol.SetTraceValue(trace)

	switch trace {
	case protocol.TraceValueOff:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case protocol.TraceValueMessage:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case protocol.TraceValueVerbose:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func applyConfig(c Config) {
	trace := traceFromConfig(c)
	c.Trace.Server = trace

	setConfig(c)
	applyTraceLevel(trace)
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
		c := GetConfig()
		if err := json.Unmarshal(result[0], &c); err != nil {
			log.Error().Err(err).Msg("failed to parse config")
		} else {
			applyConfig(c)
			log.Debug().Any("config", c).Msg("config stored")
		}
	}

	return nil
}

func ConfigChanged(_ *glsp.Context, params *protocol.DidChangeConfigurationParams) error {
	raw, err := json.Marshal(params.Settings)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal changed settings")
		return nil
	}

	c := GetConfig()
	if err := json.Unmarshal(raw, &c); err != nil {
		log.Error().Err(err).Msg("failed to parse changed settings")
		return nil
	}

	applyConfig(c)
	log.Debug().Any("config", c).Msg("config changed")

	return nil
}

func SetTrace(_ *glsp.Context, params *protocol.SetTraceParams) error {
	log.Debug().Any("params", params).Msg("SetTrace")

	c := GetConfig()
	c.Trace.Server = params.Value
	applyConfig(c)

	log.Debug().Any("trace value", params.Value).Msg("trace level set")

	return nil
}
