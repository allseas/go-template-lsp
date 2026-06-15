import * as assert from "assert";
import { after, before } from "mocha";
import * as vscode from "vscode";
import { cleanupDocument, createDocument } from "./utils";

const waitTime = 400;

async function getDiagnosticsFor(
    filename: string,
    content: string,
): Promise<vscode.Diagnostic[]> {
    const { tmplUri } = await createDocument(filename, content);
    try {
        await new Promise((resolve) => setTimeout(resolve, waitTime));
        return vscode.languages.getDiagnostics(tmplUri);
    } finally {
        cleanupDocument(tmplUri);
    }
}

function assertError(diag: vscode.Diagnostic, label = "Diagnostic") {
    assert.strictEqual(
        diag.severity,
        vscode.DiagnosticSeverity.Error,
        `${label} should be an error`,
    );
}

function assertWarning(diag: vscode.Diagnostic, label = "Diagnostic") {
    assert.strictEqual(
        diag.severity,
        vscode.DiagnosticSeverity.Warning,
        `${label} should be a warning`,
    );
}

function assertContains(diag: vscode.Diagnostic, substr: string) {
    assert.ok(
        diag.message.includes(substr),
        `Expected '${substr}' in message, got: ${diag.message}`,
    );
}

suite("Diagnostics Test Suite", () => {
    before(async () => {
        await new Promise((resolve) => setTimeout(resolve, waitTime));
    });

    after(() => {
        vscode.window.showInformationMessage("All diagnostics tests done!");
    });

    test("Diagnostics on incorrect syntax", async () => {
        const diags = await getDiagnosticsFor(
            "diagnostics-incorrect-syntax-test.tmpl",
            "{{ {}}}}}{{{}}}}\nabc\n",
        );
        assert.ok(
            diags.length >= 1,
            `Expected at least 1 diagnostic, got ${diags.length}`,
        );
        assertError(diags[0], "First diagnostic");
    });

    test("No Diagnostics on correct file", async () => {
        const diags = await getDiagnosticsFor(
            "diagnostics-correct-file-test.tmpl",
            '{{/*some comment*/}}\n\n{{ $test := 0 }}\n\n{{ .Address.Country }}\n\n{{ $test }}\n{{ $test }}{{$variable := "abc"}}{{ $variable }}',
        );
        assert.strictEqual(
            diags.length,
            0,
            `Expected no diagnostics, got ${diags.length}`,
        );
    });

    test("Diagnostics on missing closing tag", async () => {
        const diags = await getDiagnosticsFor(
            "diagnostics-missing-closing-tag-test.tmpl",
            "{{ if .Condition }}\nContent without closing tag\n",
        );
        assert.ok(
            diags.length >= 1,
            `Expected at least 1 diagnostic, got ${diags.length}`,
        );
        assertError(diags[0], "First diagnostic");
    });

    test("Diagnostics on variable redeclaration", async () => {
        const diags = await getDiagnosticsFor(
            "diagnostics-variable-redeclaration-test.tmpl",
            "{{ $variable := 0 }}\n{{ $variable := 1 }}\n",
        );
        assert.strictEqual(
            diags.length,
            1,
            `Expected 1 diagnostic, got ${diags.length}`,
        );
        assertWarning(diags[0], "First diagnostic");
        assert.strictEqual(
            diags[0].message,
            "2:4: duplicate variable declaration: $variable",
            "Diagnostic message should indicate variable redeclaration",
        );
    });

    test("Diagnostics on undefined variable usage", async () => {
        const diags = await getDiagnosticsFor(
            "diagnostics-undefined-variable-test.tmpl",
            "{{ $UndefinedVariable }}\n",
        );
        assert.strictEqual(
            diags.length,
            1,
            `Expected 1 diagnostic, got ${diags.length}`,
        );
        assertError(diags[0], "First diagnostic");
        assert.strictEqual(
            diags[0].message,
            'template: t:1:4: undefined variable "$UndefinedVariable"',
            "Diagnostic message should indicate undefined variable usage",
        );
        assert.strictEqual(diags[0].source, "text-template-support");
    });

    test("Diagnostics on empty action", async () => {
        const diags = await getDiagnosticsFor(
            "diagnostics-empty-action-test.tmpl",
            "{{ }}",
        );
        assert.strictEqual(
            diags.length,
            1,
            `Expected 1 diagnostic, got ${diags.length}`,
        );
        assertError(diags[0]);
        assertContains(diags[0], "missing value");
        assert.strictEqual(diags[0].range.start.character, 0);
        assert.strictEqual(diags[0].range.end.character, 5);
    });

    test("Diagnostics on unknown function", async () => {
        const diags = await getDiagnosticsFor(
            "diagnostics-unknown-function-test.tmpl",
            "{{ unknownFunc }}",
        );
        assert.ok(
            diags.length >= 1,
            `Expected at least 1 diagnostic, got ${diags.length}`,
        );
        assertError(diags[0]);
        assertContains(diags[0], "unknownFunc");
        assertContains(diags[0], "unsupported");
    });

    test("Diagnostics on undeclared variable (server-side)", async () => {
        const diags = await getDiagnosticsFor(
            "diagnostics-undeclared-variable-server-test.tmpl",
            "{{ $x }}\n",
        );
        assert.ok(
            diags.length >= 1,
            `Expected at least 1 diagnostic, got ${diags.length}`,
        );
        assertError(diags[0]);
        assertContains(diags[0], "$x");
    });

    test("Multiple diagnostics in one file", async () => {
        const diags = await getDiagnosticsFor(
            "diagnostics-multiple-test.tmpl",
            "{{ badOne }}\n{{ badTwo }}\n",
        );
        assert.ok(
            diags.length >= 2,
            `Expected at least 2 diagnostics, got ${diags.length}`,
        );
        const messages = diags.map((d) => d.message);
        assert.ok(
            messages.some((m) => m.includes("badOne")),
            "Expected diagnostic for badOne",
        );
        assert.ok(
            messages.some((m) => m.includes("badTwo")),
            "Expected diagnostic for badTwo",
        );
        const lines = diags.map((d) => d.range.start.line);
        assert.ok(lines.includes(0), "Expected diagnostic on line 0");
        assert.ok(lines.includes(1), "Expected diagnostic on line 1");
    });

    test("Diagnostics on incorrect range", async () => {
        const diags = await getDiagnosticsFor(
            "diagnostics-incorrect-range-test.tmpl",
            "{{ if .Condition }}\nContent without closing tag\n",
        );
        assert.ok(
            diags.length >= 1,
            `Expected at least 1 diagnostic, got ${diags.length}`,
        );
        assertError(diags[0], "First diagnostic");
        assert.ok(
            diags[0].range.start.line >= 0 &&
                diags[0].range.start.character >= 0,
            "Diagnostic range should have valid start position",
        );
        assert.ok(
            diags[0].range.end.line >= 0 && diags[0].range.end.character >= 0,
            "Diagnostic range should have valid end position",
        );
    });
});
