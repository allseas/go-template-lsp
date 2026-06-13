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

func msgParseError(text string, offset int, str string) string {
	return withPos(text, offset, "parse error: "+str)
}

func msgUnknownFunction(text string, offset int, name string) string {
	return withPos(text, offset, "unsupported function or unregistered command: "+name)
}

// withPos prepends a "line:col: " prefix to msg.
func withPos(text string, offset int, msg string) string {
	p := offsetToPosition(text, offset)
	return fmt.Sprintf("%d:%d: %s", p.Line+1, p.Character+1, msg)
}
