package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSmoke(t *testing.T) {
	t.Parallel()
	tn := TableNode{
		NodeType: 0,
		Pos:      0,
		Line:     0,
		Format:   "",
		Pipe:     nil,
		List:     nil,
	}

	require.Equal(t, "{{block (table extension)}}", tn.String())
}
