package main

import (
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// references finds and shows all references of a variable or function
func references(_ *glsp.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	text, ok := store.Get(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	offset := positionToOffset(text, params.Position)
	if !isInsideTemplate(text, offset) {
		log.Debug().
			Int("offset new ", offset).
			Msg("completion: cursor is not inside a template block, skipping")
		return nil, nil
	}
	word := getWordAtOffset(text, offset)
	if word == "" {
		log.Debug().Msg("references: no word found at current offset")
		return nil, nil
	}
	escapedWord := regexp.QuoteMeta(word)
	pattern, err := regexp.Compile(escapedWord)
	if err != nil {
		log.Error().Err(err).Str("word", word).Msg("references: failed to compile regex")
		return nil, err
	}
	var locations []protocol.Location
	lines := strings.Split(text, "\n")
	byteOffset := 0
	for lineNum, line := range lines {
		for _, loc := range pattern.FindAllStringIndex(line, -1) {
			matchOffset := byteOffset + loc[0]
			if !isInsideTemplate(text, matchOffset) {
				continue
			}
			locations = append(locations, protocol.Location{
				URI: params.TextDocument.URI,
				Range: protocol.Range{
					Start: protocol.Position{Line: uint32(lineNum), Character: uint32(utf16Len(line[:loc[0]]))},
					End:   protocol.Position{Line: uint32(lineNum), Character: uint32(utf16Len(line[:loc[1]]))},
				},
			})
		}
		byteOffset += len(line) + 1
	}
	log.Info().
		Int("count", len(locations)).
		Str("word", word).
		Msg("references: search complete")
	return locations, nil
}
