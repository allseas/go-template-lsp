// Package handlers provides handlers for LSP requests and notifications related to server configuration, including retrieving and updating settings, and managing trace levels for logging.
package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"text-template-server/types"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TraceValueMessages is the "messages" trace level. The "message" one in glsp is wrong.
const TraceValueMessages = protocol.TraceValue("messages")

// DiagnosticsSeverity represents the severity level for diagnostics of a specific error type. It allows fine-grained control over which diagnostics are reported to the client and how they are categorized (e.g., as errors, warnings, informational messages, or hints).
type DiagnosticsSeverity int

const (
	// DiagnosticSeverityDisabled means that diagnostics of this type are not reported to the client at all.
	DiagnosticSeverityDisabled DiagnosticsSeverity = iota
	// DiagnosticSeverityError means that diagnostics of this type are reported as errors.
	DiagnosticSeverityError
	// DiagnosticSeverityWarning means that diagnostics of this type are reported as warnings.
	DiagnosticSeverityWarning
	// DiagnosticSeverityInformation means that diagnostics of this type are reported as informational messages.
	DiagnosticSeverityInformation
	// DiagnosticSeverityHint means that diagnostics of this type are reported as hints.
	DiagnosticSeverityHint
)

var diagnosticsSeverityNames = map[DiagnosticsSeverity]string{
	DiagnosticSeverityDisabled:    "disabled",
	DiagnosticSeverityError:       "error",
	DiagnosticSeverityWarning:     "warning",
	DiagnosticSeverityInformation: "information",
	DiagnosticSeverityHint:        "hint",
}

// MarshalText implements encoding.TextMarshaler so DiagnosticsSeverity is serialized as a string in JSON.
func (s DiagnosticsSeverity) MarshalText() ([]byte, error) {
	if name, ok := diagnosticsSeverityNames[s]; ok {
		return []byte(name), nil
	}
	return nil, fmt.Errorf("unknown DiagnosticsSeverity: %d", int(s))
}

// UnmarshalText implements encoding.TextUnmarshaler so DiagnosticsSeverity can be deserialized from a string.
func (s *DiagnosticsSeverity) UnmarshalText(data []byte) error {
	for k, v := range diagnosticsSeverityNames {
		if v == string(data) {
			*s = k
			return nil
		}
	}
	return fmt.Errorf("unknown DiagnosticsSeverity: %q", string(data))
}

// DiagnosticsConfig controls which individual diagnostic categories are reported.
type DiagnosticsConfig map[types.ErrorType]DiagnosticsSeverity

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
			types.ErrorTypeInvalidField:       DiagnosticSeverityError,
			types.ErrorTypeInvalidFunction:    DiagnosticSeverityWarning,
			types.ErrorTypeInvalidCommand:     DiagnosticSeverityError,
			types.ErrorTypeInvalidRange:       DiagnosticSeverityError,
			types.ErrorTypeInvalidIf:          DiagnosticSeverityError,
			types.ErrorTypeInvalidWith:        DiagnosticSeverityError,
			types.ErrorUndeclaredVariable:     DiagnosticSeverityError,
			types.ErrorDoubleDeclaredVariable: DiagnosticSeverityWarning,
			types.ErrorTypeInvalidTemplateArg: DiagnosticSeverityError,
			types.ErrorArgumentNumberMismatch: DiagnosticSeverityError,
			types.ErrorUnknownType:            DiagnosticSeverityInformation,
			types.ErrorSyntaxError:            DiagnosticSeverityError,
			types.ErrorHintLoadFailure:        DiagnosticSeverityWarning,
			types.ErrorTypeUnknownRangeType:   DiagnosticSeverityWarning,
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

// ConfigChanged is an LSP notification handler that re-fetches configuration from the client
// and refreshes all open documents. It runs asynchronously to avoid deadlocking the jsonrpc2
// read loop (which would otherwise block waiting for the workspace/configuration reply).
func ConfigChanged(context *glsp.Context, _ *protocol.DidChangeConfigurationParams) error {
	if context == nil {
		return nil
	}
	go applyConfigChange(context)
	return nil
}

func applyConfigChange(ctx *glsp.Context) {
	if err := RequestConfig(ctx); err != nil {
		log.Error().Err(err).Msg("failed to request config on change")
		return
	}
	RefreshAllDocuments(ctx)
	log.Debug().Any("config", GetConfig()).Msg("config changed, documents refreshed")
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
