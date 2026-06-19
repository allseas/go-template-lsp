package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilePathToURI(t *testing.T) {
	t.Run("unix absolute path", func(t *testing.T) {
		got := FilePathToURI("/home/user/project/funcs.go")
		assert.Equal(t, "file:///home/user/project/funcs.go", got)
	})

	t.Run("windows drive path", func(t *testing.T) {
		got := FilePathToURI("C:/Users/project/funcs.go")
		assert.Equal(t, "file:///C:/Users/project/funcs.go", got)
	})

	t.Run("already slash-prefixed", func(t *testing.T) {
		got := FilePathToURI("/already/absolute.go")
		assert.Equal(t, "file:///already/absolute.go", got)
	})
}
