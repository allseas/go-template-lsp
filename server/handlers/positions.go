// Package handlers provides Language Server Protocol support for Go text/templates.
package handlers

import (
	"strings"

	"github.com/rs/zerolog/log"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// isInsideTemplate determines if a given byte offset resides within the delimiters of a template action and is not a comment.
func isInsideTemplate(text string, offset int) bool {
	if offset > len(text) || offset < 0 {
		log.Warn().
			Int("offset", offset).
			Int("text_len", len(text)).
			Msg("isInsideTemplate: offset out of bounds, clamping value")

		if offset > len(text) {
			offset = len(text)
		} else {
			offset = 0
		}
	}

	sub := text[:offset]

	lastOpen := strings.LastIndex(sub, "{{")
	if lastOpen == -1 {
		return false
	}

	lastClose := strings.LastIndex(sub, "}}")
	if lastClose > lastOpen {
		return false
	}

	if strings.HasPrefix(sub[lastOpen:], "{{/*") {
		return false
	}

	return true
}

// getWordAtOffset returns the sequence of valid identifier characters immediately preceding the given byte offset.
func getWordAtOffset(text string, offset int) string {
	if offset > len(text) || offset < 0 {
		log.Warn().
			Int("offset", offset).
			Msg("getWordAtOffset: offset out of bounds; clamping")

		if offset > len(text) {
			offset = len(text)
		} else {
			offset = 0
		}
	}

	start := offset
	for start > 0 && isWordChar(rune(text[start-1])) {
		start--
	}
	return text[start:offset]
}

// positionToOffset translates an LSP line and character position into a flat byte offset, accounting for multibyte UTF-8 characters.
func positionToOffset(text string, pos protocol.Position) int {
	line := uint32(0)
	charUTF16 := uint32(0)

	for byteOffset, r := range text {
		if line == pos.Line && charUTF16 >= pos.Character {
			return byteOffset
		}

		if r == '\n' {
			line++
			charUTF16 = 0
			continue
		}

		if line == pos.Line {
			if r > 0xFFFF {
				charUTF16 += 2
			} else {
				charUTF16++
			}
		}
	}

	log.Debug().
		Int("line", int(line)).
		Int("chars", int(charUTF16)).
		Msg("character emitted by pos")

	return len(text)
}

// isWordChar reports whether a rune is a valid character for a template variable or function name.
func isWordChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_' || c == '$'
}

func utf16Len(s string) int {
	count := 0
	for _, r := range s {
		if r > 0xFFFF {
			count += 2
		} else {
			count++
		}
	}
	return count
}
