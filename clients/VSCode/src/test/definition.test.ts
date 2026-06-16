import * as assert from "assert";
import * as fs from "fs";
import { after, before } from "mocha";
import * as path from "path";
import * as vscode from "vscode";
import { cleanupDocument, createDocument } from "./utils";

const testCasesDir = path.join(__dirname, "../../../../test/testcases");
const RESOURCE_DIR = "DefinitionTestResources";

interface DefinitionTestCase {
    name: string;
    content: string;
    vscodeOnly?: boolean;
    expected: {
        targetLine?: number;
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
                const definitions = await getDefinitions(
                    tmplUri,
                    new vscode.Position(line, character),
                );

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

    test("Go to definition on field jumps to struct field declaration", async () => {
        const { tmplUri } = await createDocument(
            `${RESOURCE_DIR}/definition-field-test.tmpl`,
            "{{/*gotype: cg/model.Order*/}}\n{{ .CustomerName }}",
        );

        try {
            // char 5 is inside "CustomerName" (after "{{ .")
            const definitions = await pollDefinitions(
                tmplUri,
                new vscode.Position(1, 5),
            );

            assert.ok(definitions, "Definitions should be returned");
            assert.ok(
                definitions.length >= 1,
                `Expected at least 1 definition, got ${definitions.length}`,
            );
            assert.ok(
                definitions[0].uri.fsPath.endsWith("model.go"),
                `Expected definition in model.go, got ${definitions[0].uri.fsPath}`,
            );
            assert.strictEqual(
                definitions[0].range.start.line,
                89,
                "CustomerName should be on line 90 (0-indexed: 89)",
            );
        } finally {
            await cleanupDocument(tmplUri);
        }
    });

    test("Go to definition on method jumps to method declaration", async () => {
        const { tmplUri } = await createDocument(
            `${RESOURCE_DIR}/definition-method-test.tmpl`,
            "{{/*gotype: cg/model.Order*/}}\n{{ .DisplayName }}",
        );

        try {
            const definitions = await pollDefinitions(
                tmplUri,
                new vscode.Position(1, 5),
            );

            assert.ok(definitions, "Definitions should be returned");
            assert.ok(
                definitions.length >= 1,
                `Expected at least 1 definition, got ${definitions.length}`,
            );
            assert.ok(
                definitions[0].uri.fsPath.endsWith("model.go"),
                `Expected definition in model.go, got ${definitions[0].uri.fsPath}`,
            );
            assert.strictEqual(
                definitions[0].range.start.line,
                98,
                "DisplayName should be on line 99 (0-indexed: 98)",
            );
        } finally {
            await cleanupDocument(tmplUri);
        }
    });

    test("Go to definition on nested field first identifier jumps to field", async () => {
        const { tmplUri } = await createDocument(
            `${RESOURCE_DIR}/definition-nested-first-test.tmpl`,
            "{{/*gotype: cg/model.Order*/}}\n{{ .Address.City }}",
        );

        try {
            // char 5 is inside "Address"
            const definitions = await pollDefinitions(
                tmplUri,
                new vscode.Position(1, 5),
            );

            assert.ok(definitions, "Definitions should be returned");
            assert.ok(
                definitions.length >= 1,
                `Expected at least 1 definition, got ${definitions.length}`,
            );
            assert.ok(
                definitions[0].uri.fsPath.endsWith("model.go"),
                `Expected definition in model.go, got ${definitions[0].uri.fsPath}`,
            );
            assert.strictEqual(
                definitions[0].range.start.line,
                91,
                "Address field should be on line 92 (0-indexed: 91)",
            );
        } finally {
            await cleanupDocument(tmplUri);
        }
    });

    test("Go to definition on nested field second identifier jumps to nested field", async () => {
        const { tmplUri } = await createDocument(
            `${RESOURCE_DIR}/definition-nested-second-test.tmpl`,
            "{{/*gotype: cg/model.Order*/}}\n{{ .Address.City }}",
        );

        try {
            // "{{ .Address." is 12 chars, so char 13 is inside "City"
            const definitions = await getDefinitions(
                tmplUri,
                new vscode.Position(1, 13),
            );

            assert.ok(definitions, "Definitions should be returned");
            assert.ok(
                definitions.length >= 1,
                `Expected at least 1 definition, got ${definitions.length}`,
            );
            assert.ok(
                definitions[0].uri.fsPath.endsWith("model.go"),
                `Expected definition in model.go, got ${definitions[0].uri.fsPath}`,
            );
            assert.strictEqual(
                definitions[0].range.start.line,
                7,
                "City field in Address should be on line 8 (0-indexed: 7)",
            );
        } finally {
            await cleanupDocument(tmplUri);
        }
    });

    test("Go to definition on unknown field returns no results", async () => {
        const { tmplUri } = await createDocument(
            `${RESOURCE_DIR}/definition-unknown-field-test.tmpl`,
            "{{/*gotype: cg/model.Order*/}}\n{{ .NonExistent }}",
        );

        try {
            const definitions = await getDefinitions(
                tmplUri,
                new vscode.Position(1, 5),
            );

            assert.ok(
                !definitions || definitions.length === 0,
                `Expected no definitions for unknown field, got ${definitions?.length}`,
            );
        } finally {
            await cleanupDocument(tmplUri);
        }
    });
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
