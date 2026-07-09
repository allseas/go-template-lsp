package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// enableAutocompletion sets EnableAutocompletion: true for the duration of the test and restores the original config afterward.
func enableAutocompletion(t *testing.T) {
	t.Helper()
	original := GetConfig()
	setConfig(Config{EnableAutocompletion: true, Trace: original.Trace})
	t.Cleanup(func() { setConfig(original) })
}

// enableHover sets EnableHover: true for the duration of the test and restores the original config afterward.
func enableHover(t *testing.T) {
	t.Helper()
	original := GetConfig()
	setConfig(Config{EnableHover: true, Trace: original.Trace})
	t.Cleanup(func() { setConfig(original) })
}

func labelsFrom(t *testing.T, resp any) []string {
	t.Helper()
	list, ok := resp.(protocol.CompletionList)
	require.True(t, ok, "response should be a CompletionList")
	var labels []string
	for _, item := range list.Items {
		labels = append(labels, item.Label)
	}
	return labels
}

// ai
func TestIsInsideTemplate(t *testing.T) {
	t.Run("inside unclosed action", func(t *testing.T) {
		assert.True(t, isInsideTemplate("{{ $x", 5))
	})

	t.Run("outside after closing braces", func(t *testing.T) {
		assert.False(t, isInsideTemplate("{{ $x }}", 8))
	})

	t.Run("inside second action after first is closed", func(t *testing.T) {
		assert.True(t, isInsideTemplate("{{ $x }}{{ $y", 13))
	})

	t.Run("comment block returns false", func(t *testing.T) {
		assert.False(t, isInsideTemplate("{{/* comment", 12))
	})

	t.Run("no template markers", func(t *testing.T) {
		assert.False(t, isInsideTemplate("plain text", 5))
	})

	t.Run("empty string", func(t *testing.T) {
		assert.False(t, isInsideTemplate("", 0))
	})

	t.Run("offset right after opening braces", func(t *testing.T) {
		assert.True(t, isInsideTemplate("{{ $x", 2))
	})
}

// ai
func TestGetWordAtOffset(t *testing.T) {
	t.Run("returns full variable name", func(t *testing.T) {
		assert.Equal(t, "$foo", getWordAtOffset("{{ $foo", 7))
	})

	t.Run("returns partial variable name", func(t *testing.T) {
		assert.Equal(t, "$fo", getWordAtOffset("{{ $foo", 6))
	})

	t.Run("returns empty string at space boundary", func(t *testing.T) {
		// offset 3 is right before '$', so no word chars precede it
		assert.Equal(t, "", getWordAtOffset("{{ $foo", 3))
	})

	t.Run("returns function name", func(t *testing.T) {
		assert.Equal(t, "len", getWordAtOffset("{{ len", 6))
	})

	t.Run("offset at zero returns empty", func(t *testing.T) {
		assert.Equal(t, "", getWordAtOffset("foo", 0))
	})
}

// ai
func TestPositionToOffset(t *testing.T) {
	text := "hello\nworld"

	t.Run("line 0 character 0", func(t *testing.T) {
		assert.Equal(t, 0, positionToOffset(text, protocol.Position{Line: 0, Character: 0}))
	})

	t.Run("line 0 mid", func(t *testing.T) {
		assert.Equal(t, 3, positionToOffset(text, protocol.Position{Line: 0, Character: 3}))
	})

	t.Run("line 1 character 0", func(t *testing.T) {
		assert.Equal(t, 6, positionToOffset(text, protocol.Position{Line: 1, Character: 0}))
	})

	t.Run("line 1 mid", func(t *testing.T) {
		assert.Equal(t, 9, positionToOffset(text, protocol.Position{Line: 1, Character: 3}))
	})

	t.Run("position beyond end of text returns len", func(t *testing.T) {
		assert.Equal(
			t,
			len(text),
			positionToOffset(text, protocol.Position{Line: 10, Character: 0}),
		)
	})
}
