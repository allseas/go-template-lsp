package handlers

import "fmt"

// Every function prefixes the message with a "line:col: " derived from the
// byte offset into text, so all server-generated diagnostics carry position
// information in the same style as parser-generated errors.

func msgUndeclaredVariable(text string, offset int, name string) string {
	return withPos(text, offset, "undeclared variable: "+name)
}

func msgDuplicateDeclaration(text string, offset int, name string) string {
	return withPos(text, offset, "duplicate variable declaration: "+name)
}

func msgUnknownFunction(text string, offset int, name string) string {
	return withPos(text, offset, "unsupported function or unregistered command: "+name)
}

func msgTypeMismatch(text string, offset int, name string) string {
	return withPos(
		text,
		offset,
		"type mismatch: function does not accept piped data of this output kind: "+name,
	)
}

func msgParseError(text string, offset int, str string) string {
	return withPos(text, offset, "parse error: "+str)
}

// withPos prepends a "line:col: " prefix to msg.
func withPos(text string, offset int, msg string) string {
	p := offsetToPosition(text, offset)
	return fmt.Sprintf("%d:%d: %s", p.Line+1, p.Character+1, msg)
}
