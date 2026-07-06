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

// maxCycleRevisits limits the number of cycles allowed
const maxCycleRevisits = 1

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
	targetType := targetTypeForArg(cur, offset, text)
	if serverTypes.IsEmptyInterface(targetType) {
		return nil
	}
	return expandedFieldItems(base, targetType, mode, prefix, wordRange)
}

// targetTypeForArg determines the type required at the cursor position based on
// the function consuming the value there. Returns AnyType when unconstrained.
func targetTypeForArg(cur serverTypes.Node, offset serverTypes.Pos, text string) types.Type {
	cmd := serverTypes.EnclosingCommand(cur)
	if cmd == nil || len(cmd.Args) == 0 {
		return serverTypes.AnyType()
	}
	if cursorPastPipe(cmd, offset, text) {
		return serverTypes.AnyType()
	}
	head := cmd.Args[0]
	// When the cursor is on (or before the end of) the command head, the value
	// this command produces flows downstream: any constraint comes from the next
	// pipe stage rather than from a parameter of this command.
	if offset <= argEnd(head) {
		return pipeForwardType(cur, cmd, offset, text)
	}
	// Data-producing heads (fields, variables, parenthesized pipes) have no
	// argument slots -- everything within the command emits into the next
	// pipe stage. Only IdentifierNode heads represent function calls that
	// consume arguments in the current stage.
	id, ok := head.(*serverTypes.IdentifierNode)
	if !ok {
		return pipeForwardType(cur, cmd, offset, text)
	}
	fn := serverTypes.GlobalFuncs()[id.Ident]
	if fn == nil {
		return serverTypes.AnyType()
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Params().Len() == 0 {
		return serverTypes.AnyType()
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
	if sig.Variadic() && paramIdx == params.Len()-1 {
		if sl, ok := paramT.Underlying().(*types.Slice); ok {
			paramT = sl.Elem()
		}
	}
	return paramT
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

// commandEnd returns the byte position just past the last argument of cmd.
func commandEnd(cmd *serverTypes.CommandNode) serverTypes.Pos {
	end := serverTypes.Pos(0)
	for _, a := range cmd.Args {
		if e := argEnd(a); e > end {
			end = e
		}
	}
	return end
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

// pipeForwardType returns the type required of the value produced by cmd,
// derived from the parameter it fills in the next pipe stage. Returns AnyType
// when unconstrained.
//
// The typed tree's command grouping cannot be trusted under partial parses:
// when cmd contains an UndefinedNode the parser may insert extra recovery
// commands between cmd and the real next stage. The source text is the only
// reliable divider: `|` is a pure structural separator, never data. So the
// next stage is the first command in pipe.Cmds whose head position lies at
// or beyond the byte offset of the first `|` after cmd's textual end.
func pipeForwardType(
	cur serverTypes.Node,
	cmd *serverTypes.CommandNode,
	_ serverTypes.Pos,
	text string,
) types.Type {
	pipe := serverTypes.EnclosingPipe(cur)
	if pipe == nil {
		return serverTypes.AnyType()
	}
	cmdIdx := -1
	for i, c := range pipe.Cmds {
		if c == cmd {
			cmdIdx = i
			break
		}
	}
	if cmdIdx < 0 || cmdIdx+1 >= len(pipe.Cmds) {
		return serverTypes.AnyType()
	}
	cmdEnd := commandEnd(cmd)
	if int(cmdEnd) > len(text) {
		return serverTypes.AnyType()
	}
	pipeRel := strings.IndexByte(text[cmdEnd:], '|')
	if pipeRel < 0 {
		return serverTypes.AnyType()
	}
	nextStageStart := cmdEnd + serverTypes.Pos(pipeRel) + 1
	var id *serverTypes.IdentifierNode
	for i := cmdIdx + 1; i < len(pipe.Cmds); i++ {
		next := pipe.Cmds[i]
		if len(next.Args) == 0 {
			continue
		}
		if next.Args[0].Position() < nextStageStart {
			continue
		}
		candidate, ok := next.Args[0].(*serverTypes.IdentifierNode)
		if !ok {
			break
		}
		id = candidate
		break
	}
	if id == nil {
		return serverTypes.AnyType()
	}
	fn := serverTypes.GlobalFuncs()[id.Ident]
	if fn == nil {
		return serverTypes.AnyType()
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Params().Len() == 0 {
		return serverTypes.AnyType()
	}
	params := sig.Params()
	last := params.Len() - 1
	paramT := params.At(last).Type()
	if sig.Variadic() {
		if sl, ok := paramT.Underlying().(*types.Slice); ok {
			paramT = sl.Elem()
		}
	}
	return paramT
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
// is convertible to targetType.
func expandedFieldItems(
	base types.Type,
	targetType types.Type,
	mode pipeChainMode,
	prefix string,
	wordRange protocol.Range,
) []protocol.CompletionItem {
	if mode == pipeChainOff || targetType == nil {
		return nil
	}
	out := []protocol.CompletionItem{}
	seen := map[string]struct{}{}
	visited := map[string]int{}
	if key := chainVisitKey(base); key != "" {
		visited[key] = 1
	}
	walkChainPaths(base, targetType, mode, prefix, nil, &out, seen, 0, wordRange, visited)
	return out
}

// chainRecurseInto reports the type to recurse into for chain expansion and a
// stable key used for cycle detection. Returns nil, "" when t has no reachable
// children worth walking.
func chainRecurseInto(t types.Type) (types.Type, string) {
	if dict, ok := t.(*serverTypes.DictType); ok && dict != nil {
		return dict, ""
	}
	named := toNamed(t)
	if named == nil {
		return nil, ""
	}
	if _, isStruct := named.Underlying().(*types.Struct); !isStruct {
		return nil, ""
	}
	return named, namedKey(named)
}

// chainVisitKey returns the cycle-detection key for t, or "" when t has none.
func chainVisitKey(t types.Type) string {
	if named := toNamed(t); named != nil {
		return namedKey(named)
	}
	return ""
}

// namedKey returns the string name of the Named type
func namedKey(n *types.Named) string {
	obj := n.Obj()
	if pkg := obj.Pkg(); pkg != nil {
		return pkg.Path() + "." + obj.Name()
	}
	return obj.Name()
}

func walkChainPaths(
	t types.Type,
	targetType types.Type,
	mode pipeChainMode,
	prefix string,
	path []string,
	out *[]protocol.CompletionItem,
	seen map[string]struct{},
	depth int,
	wordRange protocol.Range,
	visited map[string]int,
) {
	if depth >= maxChainDepth {
		return
	}
	fields, _ := collectFieldsAndMethods(t)
	for _, f := range fields {
		segs := append(append([]string{}, path...), f.Name)
		if serverTypes.TypeConvertibleTo(f.Type, targetType) {
			if depth >= 1 {
				addPathItem(out, seen, prefix, segs, f.TypeName, wordRange)
			}
			continue
		}
		child, childKey := chainRecurseInto(f.Type)
		if child == nil {
			continue
		}
		if childKey != "" && visited[childKey] > maxCycleRevisits {
			continue
		}
		if mode == pipeChainStep {
			if depth == 0 {
				if childKey != "" {
					visited[childKey]++
				}
				if hasMatchingDescendant(child, targetType, 1, visited) {
					addPathItem(out, seen, prefix, segs, f.TypeName, wordRange)
				}
				if childKey != "" {
					visited[childKey]--
				}
			}
			continue
		}
		if childKey != "" {
			visited[childKey]++
		}
		walkChainPaths(child, targetType, mode, prefix, segs, out, seen, depth+1, wordRange, visited)
		if childKey != "" {
			visited[childKey]--
		}
	}
}

// hasMatchingDescendant reports whether any nested struct field reachable from t
func hasMatchingDescendant(
	t types.Type,
	targetType types.Type,
	depth int,
	visited map[string]int,
) bool {
	if depth >= maxChainDepth {
		return false
	}
	fields, _ := collectFieldsAndMethods(t)
	for _, f := range fields {
		if serverTypes.TypeConvertibleTo(f.Type, targetType) {
			return true
		}
		child, childKey := chainRecurseInto(f.Type)
		if child == nil {
			continue
		}
		if childKey != "" && visited[childKey] > maxCycleRevisits {
			continue
		}
		if childKey != "" {
			visited[childKey]++
		}
		found := hasMatchingDescendant(child, targetType, depth+1, visited)
		if childKey != "" {
			visited[childKey]--
		}
		if found {
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
