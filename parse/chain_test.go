package parse

import (
	"slices"
	"testing"
)

// TestChainNodeFieldPos verifies that a ChainNode records the position of the
// '.' before each chained field, independent of the spacing and literals in its
// parenthesised base.
func TestChainNodeFieldPos(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantField []string
		wantPos   []Pos
	}{
		{
			name:      "parenthesised pipeline",
			input:     `{{(dict "a" .X).a.City}}`,
			wantField: []string{"a", "City"},
			wantPos:   []Pos{15, 17},
		},
		{
			name:      "extra spaces inside parentheses",
			input:     `{{ (  dict   "a"   .X ).a.City }}`,
			wantField: []string{"a", "City"},
			wantPos:   []Pos{23, 25},
		},
		{
			name:      "parentheses inside string literals",
			input:     "{{(printf \")(\" `)` .X).Len}}",
			wantField: []string{"Len"},
			wantPos:   []Pos{22},
		},
		{
			name:      "nested parenthesised chains",
			input:     `{{((dict "a" .X).a).City}}`,
			wantField: []string{"City"},
			wantPos:   []Pos{19},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := New(tt.name)
			tr.Mode = SkipFuncCheck
			if _, err := tr.Parse(tt.input, "", "", make(map[string]*Tree)); err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			action, ok := tr.Root.Nodes[0].(*ActionNode)
			if !ok {
				t.Fatalf("expected ActionNode, got %T", tr.Root.Nodes[0])
			}
			chain, ok := action.Pipe.Cmds[0].Args[0].(*ChainNode)
			if !ok {
				t.Fatalf("expected ChainNode, got %T", action.Pipe.Cmds[0].Args[0])
			}
			if !slices.Equal(chain.Field, tt.wantField) {
				t.Errorf("Field: got %q, want %q", chain.Field, tt.wantField)
			}
			if !slices.Equal(chain.FieldPos, tt.wantPos) {
				t.Errorf("FieldPos: got %v, want %v", chain.FieldPos, tt.wantPos)
			}
			for i, p := range chain.FieldPos {
				if got := tt.input[p : int(p)+1+len(chain.Field[i])]; got != "."+chain.Field[i] {
					t.Errorf("source at FieldPos[%d]: got %q, want %q", i, got, "."+chain.Field[i])
				}
			}
			if cp := chain.Copy().(*ChainNode); !slices.Equal(cp.FieldPos, chain.FieldPos) {
				t.Errorf("Copy FieldPos: got %v, want %v", cp.FieldPos, chain.FieldPos)
			}
		})
	}
}
