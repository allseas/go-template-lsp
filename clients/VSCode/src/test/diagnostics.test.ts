import * as assert from "assert";
import { after, before } from "mocha";
import * as vscode from "vscode";
import { cleanupDocument, createDocument } from "./utils";

const timeout = 300;

suite("Diagnostics Test Suite", () => {
    before(async () => {
        await new Promise((resolve) => setTimeout(resolve, timeout));
    });

    after(() => {
        vscode.window.showInformationMessage("All diagnostics tests done!");
    });

    test("Diagnostics on incorrect syntax", async () => {
        const { tmplUri } = await createDocument(
            "diagnostics-incorrect-syntax-test.tmpl",
            "{{ {}}}}}{{{}}}}\nabc\n",
        );

        try {
            await new Promise((resolve) => setTimeout(resolve, timeout));

            const diagnostics = vscode.languages.getDiagnostics(tmplUri);

            assert.ok(diagnostics, "Diagnostics should be returned");
            assert.ok(
                diagnostics.length >= 1,
                `Expected at least 1 diagnostic, got ${diagnostics.length}`,
            );
            assert.strictEqual(
                diagnostics[0].severity,
                vscode.DiagnosticSeverity.Error,
                "First diagnostic should be an error",
            );
        } finally {
            cleanupDocument(tmplUri);
        }
    });

    test("No Diagnostics on correct file", async () => {
        const { tmplUri } = await createDocument(
            "diagnostics-correct-file-test.tmpl",
            "{{/*gotype: cg/model.Order*/}}\n\n{{ $test := 0 }}\n\n{{ .Address.Country }}\n\n{{ $test }}\n{{ $test }}{{$variable := \"abc\"}}{{ $variable }}",
        );

        try {
            await new Promise((resolve) => setTimeout(resolve, timeout));

            const diagnostics = vscode.languages.getDiagnostics(tmplUri);

            assert.ok(diagnostics, "Diagnostics should be returned");
            assert.ok(
                diagnostics.length === 0,
                `Expected no diagnostics, got ${diagnostics.length}`,
            );
        } finally {
            cleanupDocument(tmplUri);
        }
    });

    test("Diagnostics on missing closing tag", async () => {
        const { tmplUri } = await createDocument(
            "diagnostics-missing-closing-tag-test.tmpl",
            "{{ if .Condition }}\nContent without closing tag\n",
        );

        try {
            await new Promise((resolve) => setTimeout(resolve, timeout));

            const diagnostics = vscode.languages.getDiagnostics(tmplUri);

            assert.ok(diagnostics, "Diagnostics should be returned");
            assert.ok(
                diagnostics.length >= 1,
                `Expected at least 1 diagnostic, got ${diagnostics.length}`,
            );
            assert.strictEqual(
                diagnostics[0].severity,
                vscode.DiagnosticSeverity.Error,
                "First diagnostic should be an error",
            );
        } finally {
            cleanupDocument(tmplUri);
        }
    });

    test("Diagnostics on variable redeclaration", async () => {
        const { tmplUri } = await createDocument(
            "diagnostics-variable-redeclaration-test.tmpl",
            "{{ $variable := 0 }}\n{{ $variable := 1 }}\n",
        );

        try {
            await new Promise((resolve) => setTimeout(resolve, timeout));

            const diagnostics = vscode.languages.getDiagnostics(tmplUri);

            assert.ok(diagnostics, "Diagnostics should be returned");
            assert.ok(
                diagnostics.length === 1,
                `Expected 1 diagnostic, got ${diagnostics.length}`,
            );
            assert.strictEqual(
                diagnostics[0].severity,
                vscode.DiagnosticSeverity.Warning,
                "First diagnostic should be a warning",
            );
            assert.strictEqual(
                diagnostics[0].message,
                "2:4: duplicate variable declaration: $variable",
                "Diagnostic message should indicate variable redeclaration",
            );
        } finally {
            cleanupDocument(tmplUri);
        }
    });

    test("Diagnostics on undefined variable usage", async () => {
        const { tmplUri } = await createDocument(
            "diagnostics-undefined-variable-test.tmpl",
            "{{ $UndefinedVariable }}\n",
        );

        try {
            await new Promise((resolve) => setTimeout(resolve, timeout));

            const diagnostics = vscode.languages.getDiagnostics(tmplUri);

            assert.ok(diagnostics, "Diagnostics should be returned");
            assert.ok(
                diagnostics.length === 1,
                `Expected 1 diagnostic, got ${diagnostics.length}`,
            );
            assert.strictEqual(
                diagnostics[0].severity,
                vscode.DiagnosticSeverity.Error,
                "First diagnostic should be an error",
            );
            assert.strictEqual(
                diagnostics[0].message,
                "template: t:1:4: undefined variable \"$UndefinedVariable\"",
                "Diagnostic message should indicate undefined variable usage",
            );
        } finally {
            cleanupDocument(tmplUri);
        }
    });

    test("Diagnostics on incorrect range", async () => {
        const { tmplUri } = await createDocument(
            "diagnostics-incorrect-range-test.tmpl",
            "{{ if .Condition }}\nContent without closing tag\n",
        );

        try {
            await new Promise((resolve) => setTimeout(resolve, timeout));

            const diagnostics = vscode.languages.getDiagnostics(tmplUri);

            assert.ok(diagnostics, "Diagnostics should be returned");
            assert.ok(
                diagnostics.length >= 1,
                `Expected at least 1 diagnostic, got ${diagnostics.length}`,
            );
            const diagnostic = diagnostics[0];
            assert.strictEqual(
                diagnostic.severity,
                vscode.DiagnosticSeverity.Error,
                "First diagnostic should be an error",
            );
            assert.ok(
                diagnostic.range.start.line >= 0 &&
                    diagnostic.range.start.character >= 0,
                "Diagnostic range should have valid start position",
            );
            assert.ok(
                diagnostic.range.end.line >= 0 &&
                    diagnostic.range.end.character >= 0,
                "Diagnostic range should have valid end position",
            );
        } finally {
            cleanupDocument(tmplUri);
        }
    });
});
