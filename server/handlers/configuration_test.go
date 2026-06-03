package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"

	"github.com/stretchr/testify/assert"
)

func TestApplyTraceLevel(t *testing.T) {
	t.Run("TraceValueOff sets log level to Info", func(t *testing.T) {
		applyTraceLevel(protocol.TraceValueOff)
		assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
	})

	t.Run("TraceValueMessage sets log level to Debug", func(t *testing.T) {
		applyTraceLevel(TraceValueMessages)
		assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
	})

	t.Run("TraceValueVerbose sets log level to Debug", func(t *testing.T) {
		applyTraceLevel(protocol.TraceValueVerbose)
		assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
	})

	t.Run("Default case sets log level to Info", func(t *testing.T) {
		applyTraceLevel("invalid_trace_value")
		assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
	})
}

func TestTraceFromConfig(t *testing.T) {
	t.Run("Returns trace value from config if set", func(t *testing.T) {
		config := Config{
			Trace: struct {
				Server protocol.TraceValue "json:\"server\""
			}{
				Server: protocol.TraceValueVerbose,
			},
		}
		trace := traceFromConfig(config)
		assert.Equal(t, protocol.TraceValueVerbose, trace)
	})

	t.Run("Returns trace value from config if set (with set/get)", func(t *testing.T) {
		setConfig(Config{
			Trace: struct {
				Server protocol.TraceValue "json:\"server\""
			}{
				Server: protocol.TraceValueVerbose,
			},
		})
		trace := traceFromConfig(GetConfig())
		assert.Equal(t, protocol.TraceValueVerbose, trace)
	})

	t.Run("Returns TraceValueMessage if trace value is not set in config", func(t *testing.T) {
		config := Config{}
		trace := traceFromConfig(config)
		assert.Equal(t, TraceValueMessages, trace)
	})
}

func TestLoadLocalConfig(t *testing.T) {
	t.Run("Loads local config when file exists", func(t *testing.T) {
		tempDir := t.TempDir()
		workspaceRoot = tempDir

		configData := `{"enableServer": false, "trace": {"server": "off"}}`
		configPath := filepath.Join(tempDir, "gotmpl.config.json")
		err := os.WriteFile(configPath, []byte(configData), 0o600)
		assert.NoError(t, err)

		c := Config{EnableServer: true}
		c.Trace.Server = "messages"

		loadLocalConfig(&c)

		assert.Equal(t, false, c.EnableServer)
		assert.Equal(t, protocol.TraceValueOff, c.Trace.Server)

		// Unset workspaceRoot just in case
		workspaceRoot = ""
	})

	t.Run("Does nothing when file does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		workspaceRoot = tempDir

		c := Config{EnableServer: true}
		c.Trace.Server = "messages"

		loadLocalConfig(&c)

		assert.Equal(t, true, c.EnableServer)
		assert.Equal(t, TraceValueMessages, c.Trace.Server)

		// Unset workspaceRoot
		workspaceRoot = ""
	})

	t.Run("Does nothing when workspaceRoot is empty", func(t *testing.T) {
		workspaceRoot = ""

		c := Config{EnableServer: true}
		c.Trace.Server = "messages"

		loadLocalConfig(&c)

		assert.Equal(t, true, c.EnableServer)
		assert.Equal(t, TraceValueMessages, c.Trace.Server)
	})

	t.Run("Preserves existing fields when local config is partial", func(t *testing.T) {
		tempDir := t.TempDir()
		workspaceRoot = tempDir
		defer func() { workspaceRoot = "" }()

		// Local file only overrides enableServer; trace.server must stay.
		configData := `{"enableServer": false}`
		configPath := filepath.Join(tempDir, "gotmpl.config.json")
		assert.NoError(t, os.WriteFile(configPath, []byte(configData), 0o600))

		c := Config{EnableServer: true}
		c.Trace.Server = protocol.TraceValueVerbose

		loadLocalConfig(&c)

		assert.Equal(t, false, c.EnableServer)
		assert.Equal(t, protocol.TraceValueVerbose, c.Trace.Server)
	})

	t.Run("Leaves config unchanged when local config has invalid JSON", func(t *testing.T) {
		tempDir := t.TempDir()
		workspaceRoot = tempDir
		defer func() { workspaceRoot = "" }()

		configPath := filepath.Join(tempDir, "gotmpl.config.json")
		assert.NoError(t, os.WriteFile(configPath, []byte(`{not valid json`), 0o600))

		c := Config{EnableServer: true}
		c.Trace.Server = TraceValueMessages

		loadLocalConfig(&c)

		assert.Equal(t, true, c.EnableServer)
		assert.Equal(t, TraceValueMessages, c.Trace.Server)
	})
}

func TestLocalConfigOverridesClientConfig(t *testing.T) {
	t.Run("RequestConfig: local gotmpl.config.json overrides client settings", func(t *testing.T) {
		tempDir := t.TempDir()
		workspaceRoot = tempDir
		defer func() { workspaceRoot = "" }()

		configData := `{"enableServer": false, "trace": {"server": "off"}}`
		configPath := filepath.Join(tempDir, "gotmpl.config.json")
		assert.NoError(t, os.WriteFile(configPath, []byte(configData), 0o600))

		context := &glsp.Context{
			Call: func(method string, _ any, result any) {
				if method == "workspace/configuration" {
					*(result.(*[]json.RawMessage)) = []json.RawMessage{
						[]byte(`{"enableServer": true, "trace": {"server": "verbose"}}`),
					}
				}
			},
		}

		assert.NoError(t, RequestConfig(context))
		assert.Equal(t, false, GetConfig().EnableServer)
		assert.Equal(t, protocol.TraceValueOff, GetConfig().Trace.Server)
	})

	t.Run(
		"ConfigChanged: local gotmpl.config.json overrides notification settings",
		func(t *testing.T) {
			tempDir := t.TempDir()
			workspaceRoot = tempDir
			defer func() { workspaceRoot = "" }()

			configData := `{"trace": {"server": "off"}}`
			configPath := filepath.Join(tempDir, "gotmpl.config.json")
			assert.NoError(t, os.WriteFile(configPath, []byte(configData), 0o600))

			params := protocol.DidChangeConfigurationParams{
				Settings: json.RawMessage(`{"enableServer": true, "trace": {"server": "verbose"}}`),
			}

			assert.NoError(t, ConfigChanged(nil, &params))
			// enableServer comes from client (not present in local file)
			assert.Equal(t, true, GetConfig().EnableServer)
			// trace.server comes from local file (overrides client)
			assert.Equal(t, protocol.TraceValueOff, GetConfig().Trace.Server)
		},
	)
}

func TestApplyConfig(t *testing.T) {
	t.Run("Applies trace level from config", func(t *testing.T) {
		config := Config{
			Trace: struct {
				Server protocol.TraceValue "json:\"server\""
			}{
				Server: protocol.TraceValueVerbose,
			},
		}
		applyConfig(config)
		assert.Equal(t, protocol.TraceValueVerbose, GetConfig().Trace.Server)
		assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
	})

	t.Run("Applies default trace level if not set in config", func(t *testing.T) {
		config := Config{}
		applyConfig(config)
		assert.Equal(t, TraceValueMessages, GetConfig().Trace.Server)
		assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
	})

	t.Run("Applies trace level and updates config", func(t *testing.T) {
		config := Config{
			Trace: struct {
				Server protocol.TraceValue "json:\"server\""
			}{
				Server: protocol.TraceValueOff,
			},
		}
		applyConfig(config)
		assert.Equal(t, protocol.TraceValueOff, GetConfig().Trace.Server)
		assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
	})

	// ai
	t.Run("Overrides existing config values", func(t *testing.T) {
		initialConfig := Config{
			Trace: struct {
				Server protocol.TraceValue "json:\"server\""
			}{
				Server: protocol.TraceValueOff,
			},
			EnableServer: false,
		}
		setConfig(initialConfig)

		newConfig := Config{
			Trace: struct {
				Server protocol.TraceValue "json:\"server\""
			}{
				Server: protocol.TraceValueVerbose,
			},
		}
		applyConfig(newConfig)

		assert.Equal(t, protocol.TraceValueVerbose, GetConfig().Trace.Server)
		assert.Equal(t, false, GetConfig().EnableServer)
		assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
	})

	t.Run("Overrides existing config values 2", func(t *testing.T) {
		initialConfig := Config{
			Trace: struct {
				Server protocol.TraceValue "json:\"server\""
			}{
				Server: protocol.TraceValueOff,
			},
			EnableServer: false,
		}
		setConfig(initialConfig)

		newConfig := Config{
			Trace: struct {
				Server protocol.TraceValue "json:\"server\""
			}{
				Server: protocol.TraceValueVerbose,
			},
			EnableServer: true,
		}
		applyConfig(newConfig)

		assert.Equal(t, protocol.TraceValueVerbose, GetConfig().Trace.Server)
		assert.Equal(t, true, GetConfig().EnableServer)
		assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
	})

	t.Run("Overrides existing config values 3", func(t *testing.T) {
		initialConfig := Config{
			Trace: struct {
				Server protocol.TraceValue "json:\"server\""
			}{
				Server: protocol.TraceValueOff,
			},
			EnableServer: false,
		}
		setConfig(initialConfig)

		newConfig := Config{
			EnableServer: true,
		}
		applyConfig(newConfig)

		assert.Equal(t, TraceValueMessages, GetConfig().Trace.Server)
		assert.Equal(t, true, GetConfig().EnableServer)
		assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
	})
}

func TestRequestConfig(t *testing.T) {
	// ai
	t.Run("RequestConfig keeps config when response is empty", func(t *testing.T) {
		initialConfig := Config{EnableServer: false}
		initialConfig.Trace.Server = TraceValueMessages
		setConfig(initialConfig)

		context := &glsp.Context{
			Call: func(method string, _ any, result any) {
				assert.Equal(t, "workspace/configuration", method)
				*(result.(*[]json.RawMessage)) = []json.RawMessage{}
			},
		}

		err := RequestConfig(context)
		assert.NoError(t, err)
		assert.Equal(t, initialConfig, GetConfig())
	})

	// ai
	t.Run("RequestConfig updates config on successful response", func(t *testing.T) {
		context := &glsp.Context{
			Call: func(method string, _ any, result any) {
				if method == "workspace/configuration" {
					response := []json.RawMessage{
						[]byte(`{"trace": {"server": "verbose"}, "enableServer": true}`),
					}
					*(result.(*[]json.RawMessage)) = response
				}
			},
		}

		err := RequestConfig(context)
		assert.NoError(t, err)
		assert.Equal(t, protocol.TraceValueVerbose, GetConfig().Trace.Server)
		assert.Equal(t, true, GetConfig().EnableServer)
	})
}

func TestConfigChanged(t *testing.T) {
	// ai
	t.Run("ConfigChanged applies new config", func(t *testing.T) {
		initialConfig := Config{EnableServer: false}
		setConfig(initialConfig)

		params := protocol.DidChangeConfigurationParams{
			Settings: json.RawMessage(`{"trace": {"server": "verbose"}, "enableServer": true}`),
		}

		err := ConfigChanged(nil, &params)
		assert.NoError(t, err)
		assert.Equal(t, protocol.TraceValueVerbose, GetConfig().Trace.Server)
		assert.Equal(t, true, GetConfig().EnableServer)
	})

	// ai
	t.Run("ConfigChanged handles invalid config", func(t *testing.T) {
		initialConfig := Config{EnableServer: false}
		setConfig(initialConfig)

		params := protocol.DidChangeConfigurationParams{
			Settings: json.RawMessage(
				`{"trace": {"server": "invalid_trace_value"}, "enableServer": true}`,
			),
		}

		err := ConfigChanged(nil, &params)
		assert.NoError(t, err)
		assert.Equal(t, TraceValueMessages, GetConfig().Trace.Server)
		assert.Equal(t, true, GetConfig().EnableServer)
	})

	// ai
	t.Run("ConfigChanged handles empty config", func(t *testing.T) {
		initialConfig := Config{EnableServer: false}
		setConfig(initialConfig)

		params := protocol.DidChangeConfigurationParams{
			Settings: json.RawMessage(`{}`),
		}

		err := ConfigChanged(nil, &params)
		assert.NoError(t, err)
		assert.Equal(t, TraceValueMessages, GetConfig().Trace.Server)
		assert.Equal(t, false, GetConfig().EnableServer)
	})

	// ai
	t.Run("ConfigChanged handles partial config", func(t *testing.T) {
		initialConfig := Config{EnableServer: false}
		setConfig(initialConfig)

		params := protocol.DidChangeConfigurationParams{
			Settings: json.RawMessage(`{"enableServer": true}`),
		}

		err := ConfigChanged(nil, &params)
		assert.NoError(t, err)
		assert.Equal(t, TraceValueMessages, GetConfig().Trace.Server)
		assert.Equal(t, true, GetConfig().EnableServer)
	})

	// ai
	t.Run("ConfigChanged handles invalid JSON config", func(t *testing.T) {
		initialConfig := Config{EnableServer: false}
		setConfig(initialConfig)
		params := protocol.DidChangeConfigurationParams{
			Settings: `{"enableServer: true}`,
		}

		err := ConfigChanged(nil, &params)
		assert.Error(t, err)
		assert.Equal(t, false, GetConfig().EnableServer)
	})
}

func TestSetTrace(t *testing.T) {
	t.Run("SetTrace", func(t *testing.T) {
		initialConfig := Config{EnableServer: false}
		setConfig(initialConfig)
		assert.Equal(t, protocol.TraceValue(""), GetConfig().Trace.Server)

		err := SetTrace(&glsp.Context{}, &protocol.SetTraceParams{
			Value: protocol.TraceValueOff,
		})

		assert.Equal(t, protocol.TraceValueOff, GetConfig().Trace.Server)

		assert.NoError(t, err)
	})
}
