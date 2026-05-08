package handlers

import (
	parse "text-template-parser"

	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// references finds and outputs all references for a selected variable or function
func references(_ *glsp.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	doc, ok := store.Get(params.TextDocument.URI)
	if !ok || doc.tree == nil {
		log.Debug().Msg("doc or tree is nil")
		return nil, nil
	}

	offset := positionToOffset(doc.text, params.Position)
	target := nodeFind(doc.tree.Root, parse.Pos(offset))
	if target == nil {
		return nil, nil
	}

	targetKey, ok := nodeKey(target)
	if !ok {
		return nil, nil
	}

	var results []protocol.Location
	includeDecl := params.Context.IncludeDeclaration

	inspect(doc.tree.Root, func(n parse.Node) bool {
		key, ok := nodeKey(n)
		if !ok || key != targetKey {
			return true
		}

		if !includeDecl && isVarDecl(n, targetKey) {
			return true
		}

		results = append(results, protocol.Location{
			URI:   params.TextDocument.URI,
			Range: nodeToRange(n, doc.text),
		})
		return true
	})

	return results, nil
}

// isVarDecl determines if a node is a definition of a variable for proper reference highlighting
func isVarDecl(n parse.Node, targetKey string) bool {
	v, ok := n.(*parse.VariableNode)
	if !ok || len(v.Ident) == 0 {
		return false
	}
	return "var:"+v.Ident[0] == targetKey
}

func nodeKey(n parse.Node) (string, bool) {
	switch node := n.(type) {
	case *parse.VariableNode:
		if len(node.Ident) > 0 {
			return "var:" + node.Ident[0], true
		}
	case *parse.FieldNode:
		return "", false
	case *parse.ChainNode:
		if v, ok := node.Node.(*parse.VariableNode); ok && len(v.Ident) > 0 {
			return "var:" + v.Ident[0], true
		}
	case *parse.IdentifierNode:
		return "id:" + node.Ident, true
	}
	return "", false
}

// inspect performs a depth-first walk over a tree, calling visitor on each node, skipping children if visitor is false
func inspect(node parse.Node, visitor func(parse.Node) bool) {
	if node == nil || !visitor(node) {
		return
	}

	switch n := node.(type) {
	case *parse.ListNode:
		for _, child := range n.Nodes {
			inspect(child, visitor)
		}
	case *parse.ActionNode:
		inspect(n.Pipe, visitor)
	case *parse.PipeNode:
		for _, decl := range n.Decl {
			inspect(decl, visitor)
		}
		for _, cmd := range n.Cmds {
			inspect(cmd, visitor)
		}
	case *parse.CommandNode:
		for _, arg := range n.Args {
			inspect(arg, visitor)
		}
	case *parse.ChainNode:
		inspect(n.Node, visitor)
	case *parse.IfNode:
		inspect(n.Pipe, visitor)
		inspect(n.List, visitor)
		if n.ElseList != nil {
			inspect(n.ElseList, visitor)
		}
	case *parse.RangeNode:
		inspect(n.Pipe, visitor)
		inspect(n.List, visitor)
		if n.ElseList != nil {
			inspect(n.ElseList, visitor)
		}
	case *parse.WithNode:
		inspect(n.Pipe, visitor)
		inspect(n.List, visitor)
		if n.ElseList != nil {
			inspect(n.ElseList, visitor)
		}
	case *parse.TemplateNode:
		inspect(n.Pipe, visitor)
	}
}

// nodeToRange converts the node into a range for use in Location()
func nodeToRange(n parse.Node, text string) protocol.Range {
	start := int(n.Position())
	length := len(n.String())
	end := start + length

	return protocol.Range{
		Start: offsetToPosition(text, start),
		End:   offsetToPosition(text, end),
	}
}

// offsetToPosition converts the integer offset into Position()
func offsetToPosition(text string, offset int) protocol.Position {
	line := uint32(0)
	charUTF16 := uint32(0)

	for i, r := range text {
		if i >= offset {
			break
		}
		if r == '\n' {
			line++
			charUTF16 = 0
			continue
		}

		if r > 0xFFFF {
			charUTF16 += 2
		} else {
			charUTF16++
		}
	}

	return protocol.Position{
		Line:      line,
		Character: charUTF16,
	}
}
