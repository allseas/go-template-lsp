//go:build !allseas

package types

import (
	"fmt"
	parse "text-template-parser"
)

// extAnalyseNode is the !allseas no-op counterpart of the dispatcher in
// node_ext_allseas.go. With no extension parse nodes compiled in, any node
// reaching here is genuinely unknown.
func extAnalyseNode(node parse.Node, _ Node, _ *analysisCtx) Node {
	panic(fmt.Sprintf("unknown node type: %T", node))
}

// extNodeChildren is the !allseas no-op counterpart; no extension typed
// nodes exist, so traversal yields no children.
func extNodeChildren(_ Node) []Node {
	return []Node{}
}

// extSetEndPos is the !allseas no-op counterpart of the boundary setter.
func extSetEndPos(_ Node, _ Pos, _ *string) {}

func extParseNodeChildren(node parse.Node) []parse.Node {
	return []parse.Node{}
}
