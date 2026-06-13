import * as assert from "assert";
import * as fs from "fs";
import { after, before } from "mocha";
import * as path from "path";
import * as vscode from "vscode";
import { cleanupDocument, createDocument } from "./utils";

const testCasesDir = path.join(__dirname, "../../../../test/testcases");

interface CompletionTestCase {
    name: string;
    content: string;
    expectedIncludes: string[];
    expectedExcludes: string[];
    expectedIncludesExactlyOnce?: string[];
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

suite("Completion Test Suite", () => {
    before(async () => {
        await new Promise((resolve) => setTimeout(resolve, 1000));
    });

    after(() => {
        vscode.window.showInformationMessage("All completion tests done!");
    });

    async function getCompletions(uri: vscode.Uri, position: vscode.Position) {
        return await vscode.commands.executeCommand<vscode.CompletionList>(
            "vscode.executeCompletionItemProvider",
            uri,
            position,
        );
    }

    function getLabels(
        completions: vscode.CompletionList | undefined,
    ): string[] {
        if (!completions) return [];
        return completions.items.map((item) =>
            typeof item.label === "string" ? item.label : item.label.label,
        );
    }

    const testCases: CompletionTestCase[] = JSON.parse(
        fs.readFileSync(path.join(testCasesDir, "completion.json"), "utf-8"),
    );

    for (const tc of testCases) {
        test(tc.name, async () => {
            const { content, line, character } = extractCursor(tc.content);
            const fileName = `completion-${tc.name.toLowerCase().replace(/[^a-z0-9]+/g, "-")}.tmpl`;
            const { tmplUri } = await createDocument(fileName, content);
            try {
                await new Promise((resolve) => setTimeout(resolve, 200));
                const list = await getCompletions(
                    tmplUri,
                    new vscode.Position(line, character),
                );
                const labels = getLabels(list);

                for (const expected of tc.expectedIncludes) {
                    assert.ok(
                        labels.includes(expected),
                        `Expected '${expected}' in completions, got: [${labels.join(", ")}]`,
                    );
                }
                for (const excluded of tc.expectedExcludes) {
                    assert.ok(
                        !labels.includes(excluded),
                        `Expected '${excluded}' to NOT be in completions`,
                    );
                }
                for (const once of tc.expectedIncludesExactlyOnce ?? []) {
                    const count = labels.filter((l) => l === once).length;
                    assert.strictEqual(
                        count,
                        1,
                        `Expected '${once}' to appear exactly once in completions, got ${count}`,
                    );
                }
            } finally {
                await cleanupDocument(tmplUri);
            }
        });
    }
});
