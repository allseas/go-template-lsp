package handlers

import (
	"github.com/rs/zerolog/log"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func ConfigChanged(context *glsp.Context, params *protocol.DidChangeConfigurationParams) error {

	log.Debug().Any("params", params).Msg("config changed")

	return nil
}

func RequestConfig(context *glsp.Context, params *protocol.ConfigurationParams) ([]any, error) {

	log.Debug().Any("params", params).Msg("requesting config")

	return []any{}, nil
}
