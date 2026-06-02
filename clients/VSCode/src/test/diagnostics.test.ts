import * as assert from "assert";
import { after, before } from "mocha";
import * as vscode from "vscode";
import { cleanupDocument, createDocument } from "./utils";

suite("Diagnostics Test Suite", () => {
    before(async () => {
        await new Promise((resolve) => setTimeout(resolve, 2000));
    });

    after(() => {
        vscode.window.showInformationMessage("All diagnostics tests done!");
    });

    test("Diagnostics on incorrect syntax", async () => {
        const { tmplUri } = await createDocument(
            "diagnostics-test.tmpl",
            "{{ {}}}}}{{{}}}}\nabc\n",
        );

        try {
            await new Promise((resolve) => setTimeout(resolve, 3000));

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
});
