//https://github.com/microsoft/vscode-extension-samples/blob/main/helloworld-test-sample/src/test/suite/extension.test.ts
import * as assert from "assert";
import { after } from "mocha";

import * as vscode from "vscode";
import { cleanupDocument, createDocument } from "./utils";
// import * as myExtension from '../extension';

suite("Snippets Test Suite", () => {
    after(() => {
        vscode.window.showInformationMessage("All snippet tests done!");
    });

    test("Snippets should be available in .tmpl files", async () => {
        const { tmplUri, document } = await createDocument(
            "snippets-test.tmpl",
            "if",
        );

        try {
            assert.strictEqual(
                document.languageId,
                "gotmpl",
                "File should be recognized as gotmpl language",
            );

            const result = await vscode.commands.executeCommand(
                "vscode.executeCompletionItemProvider",
                tmplUri,
                new vscode.Position(0, 0),
            );

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const snippets = (result as any)?.items || [];
            assert.ok(
                Array.isArray(snippets),
                "CompletionList should have items array",
            );

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            snippets.forEach((item: any, index: number) => {
                const label =
                    typeof item.label === "string"
                        ? item.label
                        : item.label?.label;
                console.log(`  ${index}: ${label}`);
            });

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const templateSnippets = (snippets as any[]).filter((item: any) => {
                const label =
                    typeof item.label === "string"
                        ? item.label
                        : item.label?.label;
                return [
                    "if",
                    "range",
                    "with",
                    "break",
                    "continue",
                    "else",
                    "var",
                    "template",
                ].includes(label as string);
            });

            assert.ok(
                templateSnippets.length > 0,
                `Template snippets should be available in .tmpl files. Got: ${snippets.length} completions, snippet labels: ${snippets
                    .slice(0, 5)
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    .map((s: any) =>
                        typeof s.label === "string" ? s.label : s.label?.label,
                    )
                    .join(", ")}`,
            );
        } finally {
            cleanupDocument(tmplUri);
        }
    });

    test("Snippets should NOT be available in non-.tmpl files", async () => {
        const { tmplUri, document } = await createDocument(
            "snippets-test.txt",
            "if",
        );

        try {
            assert.notStrictEqual(
                document.languageId,
                "gotmpl",
                "File should NOT be recognized as gotmpl language",
            );

            const result = await vscode.commands.executeCommand(
                "vscode.executeCompletionItemProvider",
                tmplUri,
                new vscode.Position(0, 0),
            );

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const completions = (result as any)?.items || [];
            assert.ok(
                Array.isArray(completions),
                "CompletionList should have items array",
            );

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const templateSnippets = (completions as any[]).filter(
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                (item: any) => {
                    const label =
                        typeof item.label === "string"
                            ? item.label
                            : item.label?.label;
                    return [
                        "if",
                        "range",
                        "with",
                        "break",
                        "continue",
                        "else",
                        "var",
                        "template",
                    ].includes(label as string);
                },
            );

            assert.strictEqual(
                templateSnippets.length,
                0,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                `Template snippets should NOT be available in non-.tmpl files. Got ${templateSnippets.length} snippets: ${templateSnippets.map((s: any) => (typeof s.label === "string" ? s.label : s.label?.label)).join(", ")}`,
            );
        } finally {
            cleanupDocument(tmplUri);
        }
    });
});
