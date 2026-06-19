package handlers

import (
	"go/types"
	"strings"

	serverTypes "text-template-server/types"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// pipeChainMode represents the configured strategy for expanding nested dot
// completions toward a downstream pipe/function constraint.
type pipeChainMode int

const (
	pipeChainOff pipeChainMode = iota
	pipeChainStep
	pipeChainFull
)

// maxChainDepth limits recursion
const maxChainDepth = 8

func currentPipeChainMode() pipeChainMode {
	switch strings.ToLower(GetConfig().PipeChainCompletion) {
	case "full":
		return pipeChainFull
	case "step":
		return pipeChainStep
	default:
		return pipeChainOff
	}
}

// chainExpansionItems wraps expandedFieldItems with the live config + context.
func chainExpansionItems(
	cur serverTypes.Node,
	base types.Type,
	prefix string,
	offset serverTypes.Pos,
	text string,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	mode := currentPipeChainMode()
	if mode == pipeChainOff || base == nil {
		return nil
	}
	targetKind := targetKindForArg(cur, offset, text)
	if targetKind == outputAny || targetKind == outputUntyped {
		return nil
	}
	return expandedFieldItems(base, targetKind, mode, prefix, wordRange)
}

// targetKindForArg determines the output kind required at the cursor position
// based on the function consuming the value there.
func targetKindForArg(cur serverTypes.Node, offset serverTypes.Pos, text string) outputKind {
	cmd := serverTypes.EnclosingCommand(cur)
	if cmd == nil || len(cmd.Args) == 0 {
		return outputAny
	}
	if cursorPastPipe(cmd, offset, text) {
		return outputAny
	}
	head := cmd.Args[0]
	// When the cursor is on (or before the end of) the command head, the value
	// this command produces flows downstream: any constraint comes from the next
	// pipe stage rather than from a parameter of this command.
	if offset <= argEnd(head) {
		return pipeForwardKind(cur, cmd)
	}
	// Otherwise the cursor fills a value-argument slot of a function command.
	id, ok := head.(*serverTypes.IdentifierNode)
	if !ok {
		return outputAny
	}
	fn := serverTypes.GlobalFuncs()[id.Ident]
	if fn == nil {
		return outputAny
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Params().Len() == 0 {
		return outputAny
	}
	params := sig.Params()
	paramIdx := cursorParamSlot(cmd, offset)
	if _, ok := cur.(*serverTypes.UndefinedNode); ok {
		paramIdx--
	}
	if paramIdx >= params.Len() {
		paramIdx = params.Len() - 1
	}
	paramT := params.At(paramIdx).Type()
	if sl, isSl := paramT.Underlying().(*types.Slice); isSl {
		paramT = sl.Elem()
	}
	return typeToOutputKind(paramT)
}

// cursorPastPipe reports whether a pipe separator appears in the source between
// the end of cmd's last argument and the cursor at offset.
func cursorPastPipe(cmd *serverTypes.CommandNode, offset serverTypes.Pos, text string) bool {
	if len(cmd.Args) == 0 {
		return false
	}
	le := argEnd(cmd.Args[len(cmd.Args)-1])
	if offset <= le || int(le) < 0 || int(offset) > len(text) {
		return false
	}
	return strings.Contains(text[le:offset], "|")
}

// argEnd returns the byte position just past arg in the source. The typed tree
// does not populate per-node end offsets
func argEnd(arg serverTypes.Node) serverTypes.Pos {
	if arg == nil {
		return 0
	}
	if e := arg.End(); e > arg.Position() {
		return e
	}
	return arg.Position() + serverTypes.Pos(len(arg.String()))
}

// cursorParamSlot returns the 0-based parameter index that the cursor at offset fills within cmd
func cursorParamSlot(cmd *serverTypes.CommandNode, offset serverTypes.Pos) int {
	slot := 0
	for i := 1; i < len(cmd.Args); i++ {
		a := cmd.Args[i]
		if offset >= a.Position() && offset <= argEnd(a) {
			return i - 1
		}
		if offset > argEnd(a) {
			slot = i
		}
	}
	return slot
}

// pipeForwardKind returns the kind required of the value produced by cmd,
// derived from the parameter it fills in the next pipe stage
func pipeForwardKind(cur serverTypes.Node, cmd *serverTypes.CommandNode) outputKind {
	pipe := serverTypes.EnclosingPipe(cur)
	if pipe == nil {
		return outputAny
	}
	cmdIdx := -1
	for i, c := range pipe.Cmds {
		if c == cmd {
			cmdIdx = i
			break
		}
	}
	if cmdIdx < 0 || cmdIdx+1 >= len(pipe.Cmds) {
		return outputAny
	}
	next := pipe.Cmds[cmdIdx+1]
	if len(next.Args) == 0 {
		return outputAny
	}
	id, ok := next.Args[0].(*serverTypes.IdentifierNode)
	if !ok {
		return outputAny
	}
	fn := serverTypes.GlobalFuncs()[id.Ident]
	if fn == nil {
		return outputAny
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Params().Len() == 0 {
		return outputAny
	}
	paramT := sig.Params().At(sig.Params().Len() - 1).Type()
	if sl, isSl := paramT.Underlying().(*types.Slice); isSl {
		paramT = sl.Elem()
	}
	return typeToOutputKind(paramT)
}

// cursorInValueSlot finds out whether the user is typing an explicit argument for a function
// it prevents cases like {{ repeat .Address.City .Address. | html }}
func cursorInValueSlot(cur serverTypes.Node, offset serverTypes.Pos, text string) bool {
	cmd := serverTypes.EnclosingCommand(cur)
	if cmd == nil || len(cmd.Args) == 0 {
		return false
	}
	if cursorPastPipe(cmd, offset, text) {
		return false
	}
	if structuralArgIdx(cur, cmd) >= 1 {
		return true
	}
	head := cmd.Args[0]
	if _, ok := head.(*serverTypes.IdentifierNode); !ok {
		return false
	}
	return offset > argEnd(head)
}

// structuralArgIdx returns the index of cur (or its ancestor that is a direct
// child of cmd) within cmd.Args, or -1 when cur is not within cmd's arguments.
func structuralArgIdx(cur serverTypes.Node, cmd *serverTypes.CommandNode) int {
	for n := cur; n != nil; n = n.Parent() {
		p := n.Parent()
		pc, ok := p.(*serverTypes.CommandNode)
		if !ok || pc != cmd {
			continue
		}
		for i, a := range cmd.Args {
			if a == n {
				return i
			}
		}
		return -1
	}
	return -1
}

// expandedFieldItems returns nested-path field completions whose leaf type
// satisfies targetKind.
func expandedFieldItems(
	base types.Type,
	targetKind outputKind,
	mode pipeChainMode,
	prefix string,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	if mode == pipeChainOff || targetKind == outputAny || targetKind == outputUntyped {
		return nil
	}
	named := toNamed(base)
	if named == nil {
		return nil
	}
	out := []protocol.CompletionItem{}
	seen := map[string]struct{}{}
	walkChainPaths(named, targetKind, mode, prefix, nil, &out, seen, 0, wordRange)
	return out
}

func walkChainPaths(
	named *types.Named,
	targetKind outputKind,
	mode pipeChainMode,
	prefix string,
	path []string,
	out *[]protocol.CompletionItem,
	seen map[string]struct{},
	depth int,
	wordRange protocol.Range,
) {
	if depth >= maxChainDepth {
		return
	}
	for _, f := range serverTypes.StructFields(named) {
		segs := append(append([]string{}, path...), f.Name)
		if basicTypeMatchesKind(f.Type, targetKind) {
			if depth >= 1 {
				addPathItem(out, seen, prefix, segs, f.TypeName, wordRange)
			}
			continue
		}
		child := toNamed(f.Type)
		if child == nil {
			continue
		}
		if _, isStruct := child.Underlying().(*types.Struct); !isStruct {
			continue
		}
		if mode == pipeChainStep {
			if depth == 0 && hasMatchingDescendant(child, targetKind, 1) {
				addPathItem(out, seen, prefix, segs, f.TypeName, wordRange)
			}
			continue
		}
		walkChainPaths(child, targetKind, mode, prefix, segs, out, seen, depth+1, wordRange)
	}
}

// hasMatchingDescendant reports whether any nested struct field reachable from
// named (without crossing methods/slices/maps) has a type matching targetKind.
func hasMatchingDescendant(named *types.Named, targetKind outputKind, depth int) bool {
	if depth >= maxChainDepth {
		return false
	}
	for _, f := range serverTypes.StructFields(named) {
		if basicTypeMatchesKind(f.Type, targetKind) {
			return true
		}
		child := toNamed(f.Type)
		if child == nil {
			continue
		}
		if _, isStruct := child.Underlying().(*types.Struct); !isStruct {
			continue
		}
		if hasMatchingDescendant(child, targetKind, depth+1) {
			return true
		}
	}
	return false
}

func addPathItem(
	out *[]protocol.CompletionItem,
	seen map[string]struct{},
	prefix string,
	segs []string,
	detail string,
	wordRange protocol.Range,
) {
	label := prefix + strings.Join(segs, ".")
	if _, dup := seen[label]; dup {
		return
	}
	seen[label] = struct{}{}
	kind := protocol.CompletionItemKindField
	sortText := "2_" + label
	item := protocol.CompletionItem{
		Label:    label,
		Kind:     &kind,
		Detail:   &detail,
		SortText: &sortText,
		TextEdit: &protocol.TextEdit{Range: wordRange, NewText: label},
	}
	*out = append(*out, item)
}
