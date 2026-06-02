// Type nodes
package types

// TODO: check license
// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"fmt"
	"go/types"
	"strconv"
	"strings"
)

var textFormat = "%s" // Changed to "%q" in tests for better error messages.

// A Node is an element in the parse tree. The interface is trivial.
// The interface contains an unexported method so that only
// types local to this package can satisfy it.
type Node interface {
	Type() NodeType
	String() string
	// Copy does a deep copy of the Node and all its components.
	// To avoid type assertions, some XxxNodes also have specialized
	// CopyXxx methods that return *XxxNode.
	Copy() Node
	Position() Pos // byte position of start of node in full original input string
	Parent() Node
	// writeTo writes the String output to the builder.
	writeTo(*strings.Builder)
	ValueType() types.Type
	IsElseList() bool
}

// NodeType identifies the type of a parse tree node.
type NodeType int

// Pos represents a byte position in the original input text from which
// this template was parsed.
type Pos int

func (p Pos) Position() Pos {
	return p
}

// Type returns itself and provides an easy default implementation
// for embedding in a Node. Embedded in all non-trivial Nodes.
func (t NodeType) Type() NodeType {
	return t
}

const (
	NodeText       NodeType = iota // Plain text.
	NodeAction                     // A non-control action such as a field evaluation.
	NodeBool                       // A boolean constant.
	NodeChain                      // A sequence of field accesses.
	NodeCommand                    // An element of a pipeline.
	NodeDot                        // The cursor, dot.
	nodeElse                       // An else action. Not added to tree.
	nodeEnd                        // An end action. Not added to tree.
	NodeField                      // A field or method name.
	NodeIdentifier                 // An identifier; always a function name.
	NodeIf                         // An if action.
	NodeList                       // A list of Nodes.
	NodeNil                        // An untyped nil constant.
	NodeNumber                     // A numerical constant.
	NodePipe                       // A pipeline of commands.
	NodeRange                      // A range action.
	NodeString                     // A string constant.
	NodeTemplate                   // A template invocation action.
	NodeVariable                   // A $ variable.
	NodeWith                       // A with action.
	NodeComment                    // A comment.
	NodeBreak                      // A break action.
	NodeContinue                   // A continue action.
	NodeUndefined                  // An undefined node
)

// Nodes.

// UndefinedNode represents a fragment of input the parser could not interpret
// while running in ParsePartial mode. The same error in Err is also appended
// to the enclosing Tree's Errors slice.
type UndefinedNode struct {
	NodeType
	Pos
	parent Node
	Err    error  // the error that caused this node to be produced
	str    string // original source text
	isElse bool   // whether this node is an else list
}

func (u *UndefinedNode) IsElseList() bool {
	return u.isElse
}

func (u *UndefinedNode) Parent() Node {
	return u.parent
}

func (u *UndefinedNode) String() string {
	var sb strings.Builder
	u.writeTo(&sb)
	return sb.String()
}

func (u *UndefinedNode) writeTo(sb *strings.Builder) {
	sb.WriteString(u.str)
}

func (u *UndefinedNode) Type() NodeType {
	return NodeUndefined
}

func (u *UndefinedNode) Copy() Node {
	return u.CopyUndefined()
}

func (u *UndefinedNode) CopyUndefined() *UndefinedNode {
	if u == nil {
		return nil
	}
	return &UndefinedNode{NodeType: NodeUndefined, Pos: u.Pos, parent: u.parent, Err: u.Err, str: u.str, isElse: u.isElse}
}

func (u *UndefinedNode) ValueType() types.Type {
	return nil
}

// ListNode holds a sequence of nodes.
type ListNode struct {
	NodeType
	Pos
	parent Node
	Nodes  []Node          // The element nodes in lexical order.
	isElse bool            // Whether this is in an else list.
	vars   []*VariableNode // Variables declared in this list, in appearance order. May be nil.
	typ    types.Type      // Resolved type of the dot in this list (set during analysis)
}

func (l *ListNode) Vars() []*VariableNode {
	return l.vars
}

func (l *ListNode) IsElseList() bool {
	return l.isElse
}

func (l *ListNode) append(n Node) {
	l.Nodes = append(l.Nodes, n)
}

func (l *ListNode) Parent() Node {
	return l.parent
}

func (l *ListNode) String() string {
	var sb strings.Builder
	l.writeTo(&sb)
	return sb.String()
}

func (l *ListNode) writeTo(sb *strings.Builder) {
	for _, n := range l.Nodes {
		n.writeTo(sb)
	}
}

func (l *ListNode) CopyList() *ListNode {
	if l == nil {
		return l
	}
	n := &ListNode{NodeType: NodeList, Pos: l.Pos, parent: l.parent, isElse: l.isElse}
	for _, elem := range l.Nodes {
		n.append(elem.Copy())
	}
	if l.vars != nil {
		n.vars = make([]*VariableNode, len(l.vars))
		for i, v := range l.vars {
			n.vars[i] = v.Copy().(*VariableNode)
		}
	}
	return n
}

func (l *ListNode) Copy() Node {
	return l.CopyList()
}

func (l *ListNode) ValueType() types.Type {
	return l.typ
}

// TextNode holds plain text.
type TextNode struct {
	NodeType
	Pos
	parent Node
	Text   []byte // The text; may span newlines.
	isElse bool   // Whether this is in an else list.
}

func (t *TextNode) IsElseList() bool {
	return t.isElse
}

func (t *TextNode) String() string {
	return fmt.Sprintf(textFormat, t.Text)
}

func (t *TextNode) writeTo(sb *strings.Builder) {
	sb.WriteString(t.String())
}

func (t *TextNode) Parent() Node {
	return t.parent
}

func (t *TextNode) Copy() Node {
	return &TextNode{NodeType: NodeText, Pos: t.Pos, parent: t.parent, Text: append([]byte{}, t.Text...), isElse: t.isElse}
}

func (t *TextNode) ValueType() types.Type {
	return nil
}

// CommentNode holds a comment.
type CommentNode struct {
	NodeType
	Pos
	parent Node
	Text   string // Comment text.
	isElse bool   // Whether this is in an else list.
}

func (c *CommentNode) IsElseList() bool {
	return c.isElse
}

func (c *CommentNode) String() string {
	var sb strings.Builder
	c.writeTo(&sb)
	return sb.String()
}

func (c *CommentNode) writeTo(sb *strings.Builder) {
	sb.WriteString("{{")
	sb.WriteString(c.Text)
	sb.WriteString("}}")
}

func (c *CommentNode) Parent() Node {
	return c.parent
}

func (c *CommentNode) Copy() Node {
	return &CommentNode{NodeType: NodeComment, Pos: c.Pos, parent: c.parent, Text: c.Text, isElse: c.isElse}
}

func (c *CommentNode) ValueType() types.Type {
	return nil
}

// PipeNode holds a pipeline with optional declaration
type PipeNode struct {
	NodeType
	Pos
	parent   Node
	Line     int             // The line number in the input. Deprecated: Kept for compatibility.
	IsAssign bool            // The variables are being assigned, not declared.
	Decl     []*VariableNode // Variables in lexical order.
	Cmds     []*CommandNode  // The commands in lexical order.
	typ      types.Type      // Resolved type of the pipe output (set during analysis)
	isElse   bool            // Whether this is in an else list.
}

func (p *PipeNode) IsElseList() bool {
	return p.isElse
}

func (p *PipeNode) append(command *CommandNode) {
	p.Cmds = append(p.Cmds, command)
}

func (p *PipeNode) String() string {
	// For some reason this was called with p == nil without a trace in the debugger.
	// Hardcoded null check to avoid panicking
	if p == nil {
		return "<nil>"
	}

	var sb strings.Builder
	p.writeTo(&sb)
	return sb.String()
}

func (p *PipeNode) writeTo(sb *strings.Builder) {
	if len(p.Decl) > 0 {
		for i, v := range p.Decl {
			if i > 0 {
				sb.WriteString(", ")
			}
			v.writeTo(sb)
		}
		if p.IsAssign {
			sb.WriteString(" = ")
		} else {
			sb.WriteString(" := ")
		}
	}
	for i, c := range p.Cmds {
		if i > 0 {
			sb.WriteString(" | ")
		}
		c.writeTo(sb)
	}
}

func (p *PipeNode) Parent() Node {
	return p.parent
}

func (p *PipeNode) CopyPipe() *PipeNode {
	if p == nil {
		return p
	}
	vars := make([]*VariableNode, len(p.Decl))
	for i, d := range p.Decl {
		vars[i] = d.Copy().(*VariableNode)
	}
	n := &PipeNode{NodeType: NodePipe, Pos: p.Pos, parent: p.parent, Line: p.Line, Decl: vars}
	n.IsAssign = p.IsAssign
	for _, c := range p.Cmds {
		n.append(c.Copy().(*CommandNode))
	}
	return n
}

func (p *PipeNode) Copy() Node {
	return p.CopyPipe()
}

func (p *PipeNode) ValueType() types.Type {
	return p.typ
}

// ActionNode holds an action (something bounded by delimiters).
// Control actions have their own nodes; ActionNode represents simple
// ones such as field evaluations and parenthesized pipelines.
type ActionNode struct {
	NodeType
	Pos
	parent Node
	Line   int       // The line number in the input. Deprecated: Kept for compatibility.
	Pipe   *PipeNode // The pipeline in the action.
	isElse bool      // Whether this is in an else list.
}

func (a *ActionNode) IsElseList() bool {
	return a.isElse
}

func (a *ActionNode) String() string {
	var sb strings.Builder
	a.writeTo(&sb)
	return sb.String()
}

func (a *ActionNode) writeTo(sb *strings.Builder) {
	sb.WriteString("{{")
	a.Pipe.writeTo(sb)
	sb.WriteString("}}")
}

func (a *ActionNode) Parent() Node {
	return a.parent
}

func (a *ActionNode) Copy() Node {
	return &ActionNode{NodeType: NodeAction, Pos: a.Pos, parent: a.parent, Line: a.Line, Pipe: a.Pipe.CopyPipe(), isElse: a.isElse}
}

func (a *ActionNode) ValueType() types.Type {
	return a.Pipe.typ
}

// CommandNode holds a command (a pipeline inside an evaluating action).
type CommandNode struct {
	NodeType
	Pos
	parent Node
	Args   []Node     // Arguments in lexical order: Identifier, field, or constant.
	typ    types.Type // Resolved type of the command result (set during analysis)
	isElse bool       // Whether this is in an else list.
}

func (c *CommandNode) IsElseList() bool {
	return c.isElse
}

func (c *CommandNode) append(arg Node) {
	c.Args = append(c.Args, arg)
}

func (c *CommandNode) String() string {
	var sb strings.Builder
	c.writeTo(&sb)
	return sb.String()
}

func (c *CommandNode) writeTo(sb *strings.Builder) {
	for i, arg := range c.Args {
		if i > 0 {
			sb.WriteByte(' ')
		}
		if arg, ok := arg.(*PipeNode); ok {
			sb.WriteByte('(')
			arg.writeTo(sb)
			sb.WriteByte(')')
			continue
		}
		arg.writeTo(sb)
	}
}

func (c *CommandNode) Copy() Node {
	if c == nil {
		return c
	}
	n := &CommandNode{NodeType: NodeCommand, Pos: c.Pos, parent: c.parent, isElse: c.isElse}
	for _, c := range c.Args {
		n.append(c.Copy())
	}
	return n
}

func (c *CommandNode) ValueType() types.Type {
	return c.typ
}

func (c *CommandNode) Parent() Node {
	return c.parent
}

// IdentifierNode holds an identifier.
type IdentifierNode struct {
	NodeType
	Pos
	tr     *Tree
	Ident  string     // The identifier's name.
	typ    types.Type // Resolved type if this is a function return type (set during analysis)
	parent Node
	isElse bool // Whether this is in an else list.
}

func (i *IdentifierNode) Parent() Node {
	return i.parent
}

func (i *IdentifierNode) Copy() Node {
	return &IdentifierNode{NodeType: NodeIdentifier, Pos: i.Pos, tr: i.tr, Ident: i.Ident, typ: i.typ, parent: i.parent, isElse: i.isElse}
}

func (i *IdentifierNode) IsElseList() bool {
	return i.isElse
}

// SetPos sets the position. [NewIdentifier] is a public method so we can't modify its signature.
// Chained for convenience.
// TODO: fix one day?
func (i *IdentifierNode) SetPos(pos Pos) *IdentifierNode {
	i.Pos = pos
	return i
}

func (i *IdentifierNode) String() string {
	return i.Ident
}

func (i *IdentifierNode) writeTo(sb *strings.Builder) {
	sb.WriteString(i.String())
}

func (i *IdentifierNode) ValueType() types.Type {
	return i.typ
}

// VariableNode holds a list of variable names, possibly with chained field
// accesses. The dollar sign is part of the (first) name.
type VariableNode struct {
	NodeType
	Pos
	Ident  []string   // Variable name and fields in lexical order.
	typ    types.Type // Resolved type of the variable (set during analysis)
	parent Node
	isElse bool // Whether this is in an else list.
}

func (v *VariableNode) IsElseList() bool {
	return v.isElse
}

func (v *VariableNode) String() string {
	var sb strings.Builder
	v.writeTo(&sb)
	return sb.String()
}

func (v *VariableNode) writeTo(sb *strings.Builder) {
	for i, id := range v.Ident {
		if i > 0 {
			sb.WriteByte('.')
		}
		sb.WriteString(id)
	}
}

func (v *VariableNode) Parent() Node {
	return v.parent
}

func (v *VariableNode) Copy() Node {
	return &VariableNode{
		parent:   v.parent,
		NodeType: NodeVariable,
		Pos:      v.Pos,
		Ident:    append([]string{}, v.Ident...),
		isElse:   v.isElse,
	}
}

func (v *VariableNode) ValueType() types.Type {
	return v.typ
}

// DotNode holds the special identifier '.'.
type DotNode struct {
	NodeType
	Pos
	parent Node
	typ    types.Type
	isElse bool // Whether this is in an else list.
}

func (d *DotNode) IsElseList() bool {
	return d.isElse
}

func (d *DotNode) Type() NodeType {
	// Override method on embedded NodeType for API compatibility.
	// TODO: Not really a problem; could change API without effect but
	// api tool complains.
	return NodeDot
}

func (d *DotNode) String() string {
	return "."
}

func (d *DotNode) writeTo(sb *strings.Builder) {
	sb.WriteString(d.String())
}
func (d *DotNode) Parent() Node {
	return d.parent
}

func (d *DotNode) Copy() Node {
	return &DotNode{NodeType: NodeDot, Pos: d.Pos, parent: d.parent, typ: d.typ, isElse: d.isElse}
}

func (d *DotNode) ValueType() types.Type {
	return d.typ
}

// NilNode holds the special identifier 'nil' representing an untyped nil constant.
type NilNode struct {
	NodeType
	Pos
	parent Node
	isElse bool // Whether this is in an else list.
}

func (n *NilNode) IsElseList() bool {
	return n.isElse
}

func (n *NilNode) Type() NodeType {
	// Override method on embedded NodeType for API compatibility.
	// TODO: Not really a problem; could change API without effect but
	// api tool complains.
	return NodeNil
}

func (n *NilNode) String() string {
	return "nil"
}

func (n *NilNode) writeTo(sb *strings.Builder) {
	sb.WriteString(n.String())
}
func (n *NilNode) Parent() Node {
	return n.parent
}

func (n *NilNode) Copy() Node {
	return &NilNode{NodeType: NodeNil, Pos: n.Pos, parent: n.parent, isElse: n.isElse}
}

func (n *NilNode) ValueType() types.Type {
	return types.Typ[types.UntypedNil]
}

// FieldNode holds a field (identifier starting with '.').
// The names may be chained ('.x.y').
// The period is dropped from each ident.
type FieldNode struct {
	NodeType
	Pos
	parent Node
	Ident  []string   // The identifiers in lexical order.
	typ    types.Type // Resolved type of the final field (set during analysis)
	isElse bool       // Whether this is in an else list.
}

func (f *FieldNode) IsElseList() bool {
	return f.isElse
}

func (f *FieldNode) String() string {
	var sb strings.Builder
	f.writeTo(&sb)
	return sb.String()
}

func (f *FieldNode) writeTo(sb *strings.Builder) {
	for _, id := range f.Ident {
		sb.WriteByte('.')
		sb.WriteString(id)
	}
}

func (f *FieldNode) Copy() Node {
	return &FieldNode{
		parent:   f.parent,
		NodeType: NodeField,
		Pos:      f.Pos,
		Ident:    append([]string{}, f.Ident...),
		isElse:   f.isElse,
	}
}

func (f *FieldNode) Parent() Node {
	return f.parent
}

func (f *FieldNode) ValueType() types.Type {
	return f.typ
}

// ChainNode holds a term followed by a chain of field accesses (identifier starting with '.').
// The names may be chained ('.x.y').
// The periods are dropped from each ident.
type ChainNode struct {
	NodeType
	Pos
	parent Node
	Node   Node
	Field  []string   // The identifiers in lexical order.
	typ    types.Type // Resolved type of the final field in chain (set during analysis)
	isElse bool       // Whether this is in an else list.
}

// Add adds the named field (which should start with a period) to the end of the chain.
func (c *ChainNode) Add(field string) {
	if len(field) == 0 || field[0] != '.' {
		panic("no dot in field")
	}
	field = field[1:] // Remove leading dot.
	if field == "" {
		panic("empty field")
	}
	c.Field = append(c.Field, field)
}

func (c *ChainNode) String() string {
	var sb strings.Builder
	c.writeTo(&sb)
	return sb.String()
}

func (c *ChainNode) writeTo(sb *strings.Builder) {
	if _, ok := c.Node.(*PipeNode); ok {
		sb.WriteByte('(')
		c.Node.writeTo(sb)
		sb.WriteByte(')')
	} else {
		c.Node.writeTo(sb)
	}
	for _, field := range c.Field {
		sb.WriteByte('.')
		sb.WriteString(field)
	}
}

func (c *ChainNode) Parent() Node {
	return c.parent
}

func (c *ChainNode) IsElseList() bool {
	return c.isElse
}

func (c *ChainNode) Copy() Node {
	return &ChainNode{
		NodeType: NodeChain,
		Pos:      c.Pos,
		Node:     c.Node,
		Field:    append([]string{}, c.Field...),
		parent:   c.parent,
		isElse:   c.isElse,
	}
}

func (c *ChainNode) ValueType() types.Type {
	return c.typ
}

// BoolNode holds a boolean constant.
type BoolNode struct {
	NodeType
	Pos
	parent Node
	True   bool // The value of the boolean constant.
	isElse bool // Whether this is in an else list.
}

func (b *BoolNode) IsElseList() bool {
	return b.isElse
}

func (b *BoolNode) Parent() Node {
	return b.parent
}

func (b *BoolNode) String() string {
	if b.True {
		return "true"
	}
	return "false"
}

func (b *BoolNode) writeTo(sb *strings.Builder) {
	sb.WriteString(b.String())
}

func (b *BoolNode) Copy() Node {
	return &BoolNode{NodeType: NodeBool, Pos: b.Pos, True: b.True, parent: b.parent, isElse: b.isElse}
}

func (b *BoolNode) ValueType() types.Type {
	return types.Typ[types.Bool]
}

// NumberNode holds a number: signed or unsigned integer, float, or complex.
// The value is parsed and stored under all the types that can represent the value.
// This simulates in a small amount of code the behavior of Go's ideal constants.
type NumberNode struct {
	NodeType
	Pos
	IsInt      bool       // Number has an integral value.
	IsUint     bool       // Number has an unsigned integral value.
	IsFloat    bool       // Number has a floating-point value.
	IsComplex  bool       // Number is complex.
	Int64      int64      // The signed integer value.
	Uint64     uint64     // The unsigned integer value.
	Float64    float64    // The floating-point value.
	Complex128 complex128 // The complex value.
	Text       string     // The original textual representation from the input.
	parent     Node
	isElse     bool // Whether this is in an else list.
}

func (n *NumberNode) IsElseList() bool {
	return n.isElse
}

func (n *NumberNode) String() string {
	return n.Text
}

func (n *NumberNode) writeTo(sb *strings.Builder) {
	sb.WriteString(n.String())
}

func (n *NumberNode) Copy() Node {
	nn := new(NumberNode)
	*nn = *n // Easy, fast, correct.
	return nn
}

func (n *NumberNode) Parent() Node {
	return n.parent
}

func (n *NumberNode) ValueType() types.Type {
	// Return a basic type based on what kind of number this is
	if n.IsComplex {
		return types.Typ[types.Complex128]
	}
	if n.IsFloat {
		return types.Typ[types.Float64]
	}
	if n.IsUint {
		return types.Typ[types.Uint64]
	}
	if n.IsInt {
		return types.Typ[types.Int64]
	}
	return nil
}

// StringNode holds a string constant. The value has been "unquoted".
type StringNode struct {
	NodeType
	Pos
	Quoted string // The original text of the string, with quotes.
	Text   string // The string, after quote processing.
	parent Node
	isElse bool // Whether this is in an else list.
}

func (s *StringNode) IsElseList() bool {
	return s.isElse
}

func (s *StringNode) String() string {
	return s.Quoted
}

func (s *StringNode) writeTo(sb *strings.Builder) {
	sb.WriteString(s.String())
}

func (s *StringNode) Parent() Node {
	return s.parent
}

func (s *StringNode) Copy() Node {
	return &StringNode{NodeType: NodeString, Pos: s.Pos, Quoted: s.Quoted, Text: s.Text, parent: s.parent, isElse: s.isElse}
}

func (s *StringNode) ValueType() types.Type {
	return types.Typ[types.String]
}

// BranchNode is the common representation of if, range, and with.
type BranchNode struct {
	NodeType
	Pos
	Line     int        // The line number in the input. Deprecated: Kept for compatibility.
	Pipe     *PipeNode  // The pipeline to be evaluated.
	List     *ListNode  // What to execute if the value is non-empty.
	ElseList *ListNode  // What to execute if the value is empty (nil if absent).
	typ      types.Type // Resolved type of the pipe output (set during analysis)
	parent   Node
	isElse   bool // Whether this is in an else list.
}

func (b *BranchNode) IsElseList() bool {
	return b.isElse
}

func (b *BranchNode) String() string {
	var sb strings.Builder
	b.writeTo(&sb)
	return sb.String()
}

func (b *BranchNode) writeTo(sb *strings.Builder) {
	name := ""
	switch b.NodeType {
	case NodeIf:
		name = "if"
	case NodeRange:
		name = "range"
	case NodeWith:
		name = "with"
	default:
		panic("unknown branch type")
	}
	sb.WriteString("{{")
	sb.WriteString(name)
	sb.WriteByte(' ')
	b.Pipe.writeTo(sb)
	sb.WriteString("}}")
	b.List.writeTo(sb)
	if b.ElseList != nil {
		sb.WriteString("{{else}}")
		b.ElseList.writeTo(sb)
	}
	sb.WriteString("{{end}}")
}

func (b *BranchNode) Parent() Node {
	return b.parent
}

func (b *BranchNode) Copy() Node {
	switch b.NodeType {
	case NodeIf:
		return &IfNode{BranchNode: BranchNode{NodeType: NodeIf, Pos: b.Pos, Line: b.Line, Pipe: b.Pipe, List: b.List, ElseList: b.ElseList, typ: b.typ, parent: b.parent, isElse: b.isElse}}
	case NodeRange:
		return &RangeNode{BranchNode: BranchNode{NodeType: NodeRange, Pos: b.Pos, Line: b.Line, Pipe: b.Pipe, List: b.List, ElseList: b.ElseList, typ: b.typ, parent: b.parent, isElse: b.isElse}}
	case NodeWith:
		return &WithNode{BranchNode: BranchNode{NodeType: NodeWith, Pos: b.Pos, Line: b.Line, Pipe: b.Pipe, List: b.List, ElseList: b.ElseList, typ: b.typ, parent: b.parent, isElse: b.isElse}}
	default:
		panic("unknown branch type")
	}
}

func (b *BranchNode) ValueType() types.Type {
	return b.typ
}

// IfNode represents an {{if}} action and its commands.
type IfNode struct {
	BranchNode
}

func (i *IfNode) Copy() Node {
	return &IfNode{BranchNode: BranchNode{NodeType: NodeIf, Pos: i.Pos, Line: i.Line, Pipe: i.Pipe.CopyPipe(), List: i.List.CopyList(), ElseList: i.ElseList.CopyList(), typ: i.typ, parent: i.parent, isElse: i.isElse}}
}

func (i *IfNode) ValueType() types.Type {
	return i.typ
}

// BreakNode represents a {{break}} action.
type BreakNode struct {
	NodeType
	Pos
	Line   int
	parent Node
	isElse bool // Whether this is in an else list.
}

func (b *BreakNode) Copy() Node {
	return &BreakNode{NodeType: NodeBreak, Pos: b.Pos, Line: b.Line, parent: b.parent, isElse: b.isElse}
}
func (b *BreakNode) String() string              { return "{{break}}" }
func (b *BreakNode) Parent() Node                { return b.parent }
func (b *BreakNode) writeTo(sb *strings.Builder) { sb.WriteString("{{break}}") }
func (b *BreakNode) ValueType() types.Type       { return nil }
func (b *BreakNode) IsElseList() bool            { return b.isElse }

// ContinueNode represents a {{continue}} action.
type ContinueNode struct {
	parent Node
	NodeType
	Pos
	Line   int
	isElse bool // Whether this is in an else list.
}

func (c *ContinueNode) Copy() Node {
	return &ContinueNode{NodeType: NodeContinue, Pos: c.Pos, Line: c.Line, parent: c.parent, isElse: c.isElse}
}
func (c *ContinueNode) String() string              { return "{{continue}}" }
func (c *ContinueNode) Parent() Node                { return c.parent }
func (c *ContinueNode) writeTo(sb *strings.Builder) { sb.WriteString("{{continue}}") }
func (c *ContinueNode) ValueType() types.Type       { return nil }
func (c *ContinueNode) IsElseList() bool            { return c.isElse }

// RangeNode represents a {{range}} action and its commands.
type RangeNode struct {
	BranchNode
}

func (r *RangeNode) Copy() Node {
	return &RangeNode{BranchNode: BranchNode{NodeType: NodeRange, Pos: r.Pos, Line: r.Line, Pipe: r.Pipe.CopyPipe(), List: r.List.CopyList(), ElseList: r.ElseList.CopyList(), typ: r.typ, parent: r.parent, isElse: r.isElse}}
}

func (r *RangeNode) ValueType() types.Type {
	return r.typ
}

// WithNode represents a {{with}} action and its commands.
type WithNode struct {
	BranchNode
}

func (w *WithNode) Copy() Node {
	return &WithNode{BranchNode: BranchNode{NodeType: NodeWith, Pos: w.Pos, Line: w.Line, Pipe: w.Pipe.CopyPipe(), List: w.List.CopyList(), ElseList: w.ElseList.CopyList(), typ: w.typ, parent: w.parent, isElse: w.isElse}}
}

func (w *WithNode) ValueType() types.Type {
	return w.typ
}

// TemplateNode represents a {{template}} action.
type TemplateNode struct {
	NodeType
	Pos
	parent Node
	Line   int       // The line number in the input. Deprecated: Kept for compatibility.
	Name   string    // The name of the template (unquoted).
	Pipe   *PipeNode // The command to evaluate as dot for the template.
	isElse bool      // Whether this is in an else list.
}

func (t *TemplateNode) IsElseList() bool {
	return t.isElse
}

func (t *TemplateNode) String() string {
	var sb strings.Builder
	t.writeTo(&sb)
	return sb.String()
}

func (t *TemplateNode) writeTo(sb *strings.Builder) {
	sb.WriteString("{{template ")
	sb.WriteString(strconv.Quote(t.Name))
	if t.Pipe != nil {
		sb.WriteByte(' ')
		t.Pipe.writeTo(sb)
	}
	sb.WriteString("}}")
}

func (t *TemplateNode) Parent() Node {
	return t.parent
}

func (t *TemplateNode) Copy() Node {
	return &TemplateNode{NodeType: NodeTemplate, Pos: t.Pos, Line: t.Line, Name: t.Name, Pipe: t.Pipe.CopyPipe(), parent: t.parent, isElse: t.isElse}
}

func (t *TemplateNode) ValueType() types.Type {
	return nil
}
