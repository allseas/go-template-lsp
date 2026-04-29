package main

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestCompletionLogic(t *testing.T) {
	uri := "file:///scope-test.tmpl"

	content := `
{{ $top := . }}
{{ range $i, $v := .Items }}
	{{ $he
{{ end }}
{{ $late := . }}
`

	store.Set(uri, content)
	defer store.Remove(uri)

	params := &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 3, Character: 5},
		},
	}

	resp, err := completion(nil, params)
	if err != nil {
		t.Fatalf("Completion handler failed: %v", err)
	}

	list := resp.(protocol.CompletionList)

	found := map[string]bool{}
	for _, item := range list.Items {
		found[item.Label] = true
	}

	if !found["$top"] {
		t.Error("missing $top")
	}

	if !found["$i"] {
		t.Error("missing $i (range variable)")
	}
	if !found["$v"] {
		t.Error("missing $v (range variable)")
	}

	if found["$late"] {
		t.Error("unexpected $late in completion (out of scope)")
	}

	if !found["len"] {
		t.Error("missing function len")
	}
	if !found["printf"] {
		t.Error("missing function printf")
	}
}
