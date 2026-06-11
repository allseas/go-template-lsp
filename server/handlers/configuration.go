// Package handlers provides handlers for LSP requests and notifications related to server configuration, including retrieving and updating settings, and managing trace levels for logging.
package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TraceValueMessages is the "messages" trace level. The "message" one in glsp is wrong.
const TraceValueMessages = protocol.TraceValue("messages")

// DiagnosticsConfig controls which individual diagnostic categories are reported.
type DiagnosticsConfig struct {
	SyntaxError           bool `json:"syntaxError"`
	VariableRedeclaration bool `json:"variableRedeclaration"`
	IncorrectFunction     bool `json:"incorrectFunction"`
}

// Config represents the server's configuration settings. It is designed to be updated based on client settings and can be safely accessed across concurrent requests.
type Config struct {
	EnableHover          bool              `json:"enableHover"`
	EnableDefinition     bool              `json:"enableDefinition"`
	EnableDiagnostics    bool              `json:"enableDiagnostics"`
	Diagnostics          DiagnosticsConfig `json:"diagnostics"`
	EnableAutocompletion bool              `json:"enableAutocompletion"`
	Trace                struct {
		Server protocol.TraceValue `json:"server"`
	} `json:"trace"`
}

var (
	currentConfig = Config{
		EnableHover:       true,
		EnableDefinition:  true,
		EnableDiagnostics: true,
		Diagnostics: DiagnosticsConfig{
			SyntaxError:           true,
			VariableRedeclaration: true,
			IncorrectFunction:     true,
		},
		EnableAutocompletion: true,
	}
	configMu sync.RWMutex
)

// GetConfig safely retrieves the current configuration settings. It uses a read lock to ensure thread-safe access to the global configuration variable.
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

		valid := []protocol.TraceValue{
			protocol.TraceValueOff,
			TraceValueMessages,
			protocol.TraceValueVerbose,
		}

		if !slices.Contains(valid, c.Trace.Server) {
			return TraceValueMessages
		}

		return c.Trace.Server
	}

	return TraceValueMessages
}

func loadLocalConfig(c *Config) {
	if WorkspaceRoot == "" {
		return
	}
	configPath := filepath.Join(WorkspaceRoot, "gotmpl.config.json")
	data, err := os.ReadFile(configPath) // #nosec G304 -- path is workspaceRoot + fixed filename
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error().Err(err).Msg("failed to read local config file")
		}
		return
	}
	if err := json.Unmarshal(data, c); err != nil {
		log.Error().Err(err).Msg("failed to parse local config file")
	} else {
		log.Debug().Any("config", *c).Msg("local config applied from gotmpl.config.json")
	}
}

func applyTraceLevel(trace protocol.TraceValue) {
	protocol.SetTraceValue(trace)

	switch trace {
	case protocol.TraceValueOff:
		log.Warn().Msg("trace is off, setting log level to info")
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case TraceValueMessages:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case protocol.TraceValueVerbose:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	default:
		log.Error().Any("trace", trace).Msg("default, setting log level to debug")
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func applyConfig(c Config) {
	trace := traceFromConfig(c)
	c.Trace.Server = trace

	setConfig(c)
	applyTraceLevel(trace)
}

// RequestConfig is an LSP request that retrieves the current configuration settings from the client. It sends a "workspace/configuration" request to the client, parses the response, and updates the server's configuration accordingly.
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

	c := GetConfig()
	if len(result) > 0 {
		if err := json.Unmarshal(result[0], &c); err != nil {
			log.Error().Err(err).Msg("failed to parse config")
		} else {
			log.Debug().Any("config", c).Msg("config stored from client")
		}
	}

	loadLocalConfig(&c)
	applyConfig(c)

	return nil
}

// ConfigChanged is an LSP notification handler that updates the server's configuration when the client sends new settings. It parses the incoming configuration, updates the global configuration, and applies any necessary changes.
func ConfigChanged(_ *glsp.Context, params *protocol.DidChangeConfigurationParams) error {
	raw, err := json.Marshal(params.Settings)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal changed settings")
		return err
	}

	c := GetConfig()
	if err := json.Unmarshal(raw, &c); err != nil {
		log.Error().Err(err).Msg("failed to parse changed settings")
		return err
	}

	loadLocalConfig(&c)
	applyConfig(c)
	log.Debug().Any("config", c).Msg("config changed")

	return nil
}

// SetTrace is an LSP request handler that updates the server's trace level based on client requests. It updates the global configuration and applies the new trace level immediately.
func SetTrace(_ *glsp.Context, params *protocol.SetTraceParams) error {
	log.Debug().Any("params", params).Msg("SetTrace")

	c := GetConfig()
	c.Trace.Server = params.Value
	applyConfig(c)

	log.Debug().Any("trace value", params.Value).Msg("trace level set")

	return nil
}
