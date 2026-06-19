import * as assert from "assert";
import * as fs from "fs";
import { after, before } from "mocha";
import * as path from "path";
import * as vscode from "vscode";
import { cleanupDocument, createDocument } from "./utils";

const testCasesDir = path.join(__dirname, "../../../../test/testcases");

interface DefinitionTestCase {
    name: string;
    content: string;
    vscodeOnly?: boolean;
    poll?: boolean;
    expected: {
        targetLine?: number;
        targetFile?: string;
        count?: number;
        minCount?: number;
        noResult?: boolean;
    };
}

function extractCursor(content: string): {
    content: string;
    line: number;
    character: number;
} {
    const marker = "<cursor>";
    const idx = content.indexOf(marker);
    if (idx === -1) throw new Error("No <cursor> marker found in content");
    const before = content.slice(0, idx);
    const lines = before.split("\n");
    const line = lines.length - 1;
    const character = lines[lines.length - 1].length;
    return {
        content: content.slice(0, idx) + content.slice(idx + marker.length),
        line,
        character,
    };
}

suite("Definition Test Suite", () => {
    before(async () => {
        await new Promise((resolve) => setTimeout(resolve, 1000));
    });

    after(() => {
        vscode.window.showInformationMessage("All definition tests done!");
    });

    const testCases: DefinitionTestCase[] = JSON.parse(
        fs.readFileSync(path.join(testCasesDir, "definition.json"), "utf-8"),
    );

    for (const tc of testCases) {
        test(tc.name, async () => {
            const { content, line, character } = extractCursor(tc.content);
            const fileName = `definition-${tc.name.toLowerCase().replace(/[^a-z0-9]+/g, "-")}.tmpl`;
            const { tmplUri } = await createDocument(fileName, content);
            try {
                const position = new vscode.Position(line, character);
                const definitions = tc.poll
                    ? await pollDefinitions(tmplUri, position)
                    : await getDefinitions(tmplUri, position);

                if (tc.expected.noResult) {
                    assert.ok(
                        !definitions || definitions.length === 0,
                        `Expected no definitions, got ${definitions?.length}`,
                    );
                    return;
                }

                assert.ok(definitions, "Definitions should be returned");

                if (tc.expected.count !== undefined) {
                    assert.strictEqual(
                        definitions.length,
                        tc.expected.count,
                        `Expected ${tc.expected.count} definitions, got ${definitions.length}`,
                    );
                }
                if (tc.expected.minCount !== undefined) {
                    assert.ok(
                        definitions.length >= tc.expected.minCount,
                        `Expected at least ${tc.expected.minCount} definitions, got ${definitions.length}`,
                    );
                }
                if (tc.expected.targetFile !== undefined) {
                    assert.ok(
                        definitions[0].uri.fsPath.endsWith(
                            tc.expected.targetFile,
                        ),
                        `Expected definition in ${tc.expected.targetFile}, got ${definitions[0].uri.fsPath}`,
                    );
                }
                if (tc.expected.targetLine !== undefined) {
                    assert.strictEqual(
                        definitions[0].range.start.line,
                        tc.expected.targetLine,
                        `Expected definition on line ${tc.expected.targetLine}`,
                    );
                }
            } finally {
                cleanupDocument(tmplUri);
            }
        });
    }
});

async function getDefinitions(tmplUri: vscode.Uri, pos: vscode.Position) {
    return await vscode.commands.executeCommand<vscode.Location[]>(
        "vscode.executeDefinitionProvider",
        tmplUri,
        pos,
    );
}

/** Polls until the definition provider returns at least one result or the timeout expires. */
async function pollDefinitions(
    tmplUri: vscode.Uri,
    pos: vscode.Position,
    timeoutMs = 10000,
    intervalMs = 500,
): Promise<vscode.Location[]> {
    const deadline = Date.now() + timeoutMs;
    while (Date.now() < deadline) {
        const result = await getDefinitions(tmplUri, pos);
        if (result && result.length > 0) {
            return result;
        }
        await new Promise((resolve) => setTimeout(resolve, intervalMs));
    }
    return [];
}
