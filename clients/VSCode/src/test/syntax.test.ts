import * as assert from "assert";
import { after, before } from "mocha";
import * as vscode from "vscode";
import * as vsctm from "vscode-textmate";
import { getGrammar, getScopes, assertScope } from "./utils";

suite("Syntax Highlighting Test Suite", () => {
    let grammar: vsctm.IGrammar;

    before(async () => {
        grammar = await getGrammar();
    });

    after(() => {
        vscode.window.showInformationMessage("All syntax highlighting tests done!");
    });

    test("Action delimiters are highlighted", () => {
        const line = "{{ .Foo }}";
        assertScope(getScopes(grammar, line, 0), "punctuation.definition.embedded.begin.gotmpl");
        assertScope(getScopes(grammar, line, 8), "punctuation.definition.embedded.end.gotmpl");
    });

    test("Trim marker delimiters are highlighted", () => {
        const line = "{{- .Foo -}}";
        assertScope(getScopes(grammar, line, 0), "punctuation.definition.embedded.begin.gotmpl");
        assertScope(getScopes(grammar, line, 9), "punctuation.definition.embedded.end.gotmpl");
    });

    test("Keywords are highlighted", () => {
        const keywords = ["if", "else", "end", "range", "with", "block", "define", "template", "break", "continue"];
        for (const keyword of keywords) {
            assertScope(
                getScopes(grammar, `{{ ${keyword} }}`, 3),
                "keyword.control.gotmpl",
            );
        }
    });

    test("Comment is highlighted", () => {
        assertScope(
            getScopes(grammar, "{{/* a comment */}}", 0),
            "comment.block.gotmpl",
        );
    });

    test("Trimmed comment is highlighted", () => {
        assertScope(
            getScopes(grammar, "{{- /* a comment */ -}}", 0),
            "comment.block.gotmpl",
        );
    });

    test("Variable declaration is highlighted", () => {
        const line = "{{ $name := 0 }}";
        assertScope(getScopes(grammar, line, 3), "variable.other.gotmpl");
        assertScope(getScopes(grammar, line, 9), "keyword.operator.assignment.gotmpl");
    });

    test("Variable assignment is highlighted", () => {
        const line = "{{ $x = 1 }}";
        assertScope(getScopes(grammar, line, 3), "variable.other.gotmpl");
        assertScope(getScopes(grammar, line, 6), "keyword.operator.assignment.gotmpl");
    });

    test("Variable reference is highlighted", () => {
        assertScope(getScopes(grammar, "{{ $myVar }}", 3), "variable.other.gotmpl");
    });

    test("Bare dollar sign is highlighted as variable", () => {
        assertScope(getScopes(grammar, "{{ $ }}", 3), "variable.other.gotmpl");
    });

    test("Standalone dot is highlighted", () => {
        assertScope(getScopes(grammar, "{{ . }}", 3), "variable.language.dot.gotmpl");
    });

    test("Field access is highlighted", () => {
        assertScope(getScopes(grammar, "{{ .Name }}", 3), "variable.other.member.gotmpl");
    });

    test("Builtin functions are highlighted", () => {
        const builtins = ["and", "call", "html", "index", "slice", "js", "len", "not", "or", "print", "printf", "println", "urlquery", "eq", "ne", "lt", "le", "gt", "ge"];
        for (const builtin of builtins) {
            assertScope(
                getScopes(grammar, `{{ ${builtin} }}`, 3),
                "support.function.gotmpl",
            );
        }
    });

    test("Pipe operator is highlighted", () => {
        assertScope(getScopes(grammar, "{{ .Name | html }}", 9), "keyword.operator.pipe.gotmpl");
    });

    test("Boolean true is highlighted", () => {
        assertScope(getScopes(grammar, "{{ true }}", 3), "constant.language.boolean.gotmpl");
    });

    test("Boolean false is highlighted", () => {
        assertScope(getScopes(grammar, "{{ false }}", 3), "constant.language.boolean.gotmpl");
    });

    test("Nil is highlighted", () => {
        assertScope(getScopes(grammar, "{{ nil }}", 3), "constant.language.nil.gotmpl");
    });

    test("Integer number is highlighted", () => {
        assertScope(getScopes(grammar, "{{ 42 }}", 3), "constant.numeric.gotmpl");
    });

    test("Float number is highlighted", () => {
        assertScope(getScopes(grammar, "{{ 3.14 }}", 3), "constant.numeric.gotmpl");
    });

    test("Hex number is highlighted", () => {
        assertScope(getScopes(grammar, "{{ 0xFF }}", 3), "constant.numeric.gotmpl");
    });

    test("Double-quoted string is highlighted", () => {
        assertScope(getScopes(grammar, '{{ "hello" }}', 3), "string.quoted.double.gotmpl");
    });

    test("Raw string is highlighted", () => {
        assertScope(getScopes(grammar, "{{ `raw` }}", 3), "string.quoted.other.raw.gotmpl");
    });

    test("Char literal is highlighted", () => {
        assertScope(getScopes(grammar, "{{ 'a' }}", 3), "string.quoted.single.gotmpl");
    });

    test("Escape sequence inside string is highlighted", () => {
        assertScope(getScopes(grammar, '{{ "\\n" }}', 4), "constant.character.escape.gotmpl");
    });

    test("Template name after define is highlighted", () => {
        assertScope(
            getScopes(grammar, '{{ define "myTemplate" }}', 10),
            "entity.name.function.gotmpl",
        );
    });

    test("Template name after template is highlighted", () => {
        assertScope(
            getScopes(grammar, '{{ template "myTemplate" }}', 12),
            "entity.name.function.gotmpl",
        );
    });

    test("Opening parenthesis is highlighted", () => {
        assertScope(
            getScopes(grammar, "{{ if (eq .A .B) }}", 6),
            "punctuation.section.parens.begin.gotmpl",
        );
    });

    test("Closing parenthesis is highlighted", () => {
        assertScope(
            getScopes(grammar, "{{ if (eq .A .B) }}", 15),
            "punctuation.section.parens.end.gotmpl",
        );
    });

    test("Plain text outside actions has no gotmpl token scopes", () => {
        const scopes = getScopes(grammar, "hello world", 0);
        assert.ok(
            !scopes.some((s) => s.includes("gotmpl") && s !== "source.gotmpl"),
            `Plain text should not have gotmpl-specific scopes, got: [${scopes.join(", ")}]`,
        );
    });

    test("Dot inside range block is highlighted across lines", () => {
        let ruleStack = vsctm.INITIAL;
        const line1 = grammar.tokenizeLine("{{- range .Items }}", ruleStack);
        ruleStack = line1.ruleStack;
        const line2 = grammar.tokenizeLine("{{ . }}", ruleStack);
        const scopes = (() => {
            for (const token of line2.tokens) {
                if (token.startIndex <= 3 && 3 < token.endIndex) return token.scopes;
            }
            return [];
        })();
        assertScope(scopes, "variable.language.dot.gotmpl");
    });
});