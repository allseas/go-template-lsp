import * as assert from "assert";
import * as fs from "fs";
import { after } from "mocha";
import * as path from "path";
import * as vscode from "vscode";
import { cleanupDocument, createDocument } from "./utils";

const testCasesDir = path.join(__dirname, "../../../../test/testcases");

interface DiagExpected {
    index: number;
    severity: "error" | "warning";
    message?: string;
    messageContains?: string;
    source?: string;
    rangeStart?: { line: number; character: number };
    rangeEnd?: { line: number; character: number };
    rangeStartLine?: number;
}

interface DiagnosticsTestCase {
    name: string;
    content: string;
    expected: {
        count?: number;
        minCount?: number;
        diagnostics?: DiagExpected[];
    };
}

async function getDiagnosticsFor(
    filename: string,
    content: string,
): Promise<vscode.Diagnostic[]> {
    const { tmplUri } = await createDocument(filename, content);
    try {
        return vscode.languages.getDiagnostics(tmplUri);
    } finally {
        cleanupDocument(tmplUri);
    }
}

suite("Diagnostics Test Suite", () => {
    after(() => {
        vscode.window.showInformationMessage("All diagnostics tests done!");
    });

    const testCases: DiagnosticsTestCase[] = JSON.parse(
        fs.readFileSync(path.join(testCasesDir, "diagnostics.json"), "utf-8"),
    );

    for (const tc of testCases) {
        test(tc.name, async () => {
            const fileName = `diagnostics-${tc.name.toLowerCase().replace(/[^a-z0-9]+/g, "-")}.tmpl`;
            const diags = await getDiagnosticsFor(fileName, tc.content);

            if (tc.expected.count !== undefined) {
                assert.strictEqual(
                    diags.length,
                    tc.expected.count,
                    `Expected ${tc.expected.count} diagnostics, got ${diags.length}`,
                );
            }
            if (tc.expected.minCount !== undefined) {
                assert.ok(
                    diags.length >= tc.expected.minCount,
                    `Expected at least ${tc.expected.minCount} diagnostics, got ${diags.length}`,
                );
            }

            for (const expected of tc.expected.diagnostics ?? []) {
                const diag = diags[expected.index];
                assert.ok(
                    diag,
                    `Expected diagnostic at index ${expected.index}`,
                );

                if (expected.severity === "error") {
                    assert.strictEqual(
                        diag.severity,
                        vscode.DiagnosticSeverity.Error,
                        `Diagnostic[${expected.index}] should be an error`,
                    );
                } else if (expected.severity === "warning") {
                    assert.strictEqual(
                        diag.severity,
                        vscode.DiagnosticSeverity.Warning,
                        `Diagnostic[${expected.index}] should be a warning`,
                    );
                }
                if (expected.message !== undefined) {
                    assert.strictEqual(diag.message, expected.message);
                }
                if (expected.messageContains !== undefined) {
                    assert.ok(
                        diag.message.includes(expected.messageContains),
                        `Expected '${expected.messageContains}' in message, got: ${diag.message}`,
                    );
                }
                if (expected.source !== undefined) {
                    assert.strictEqual(diag.source, expected.source);
                }
                if (expected.rangeStart !== undefined) {
                    assert.strictEqual(
                        diag.range.start.line,
                        expected.rangeStart.line,
                    );
                    assert.strictEqual(
                        diag.range.start.character,
                        expected.rangeStart.character,
                    );
                }
                if (expected.rangeEnd !== undefined) {
                    assert.strictEqual(
                        diag.range.end.line,
                        expected.rangeEnd.line,
                    );
                    assert.strictEqual(
                        diag.range.end.character,
                        expected.rangeEnd.character,
                    );
                }
                if (expected.rangeStartLine !== undefined) {
                    assert.strictEqual(
                        diag.range.start.line,
                        expected.rangeStartLine,
                    );
                }
            }
        });
    }
});
