import * as assert from "assert";
import { after, before } from "mocha";
import * as vscode from "vscode";
import { cleanupDocument, createDocument } from "./utils";

const waitTime = 300;

suite("Definition Test Suite", () => {
    before(async () => {
        await new Promise((resolve) => setTimeout(resolve, waitTime));
    });

    after(() => {
        vscode.window.showInformationMessage("All definition tests done!");
    });

    test("Go to definition on variable usage jumps to declaration", async () => {
        const { tmplUri } = await createDocument(
            "definition-test.tmpl",
            "{{ $test := 0 }}\n{{ $test }}",
        );

        try {
            await new Promise((resolve) => setTimeout(resolve, waitTime));

            const definitions = await vscode.commands.executeCommand<
                vscode.Location[]
            >(
                "vscode.executeDefinitionProvider",
                tmplUri,
                new vscode.Position(1, 4),
            );

            assert.ok(definitions, "Definitions should be returned");
            assert.ok(
                definitions.length >= 1,
                `Expected at least 1 definition, got ${definitions.length}`,
            );
            assert.strictEqual(
                definitions[0].range.start.line,
                0,
                "Definition should be on line 0",
            );
        } finally {
            cleanupDocument(tmplUri);
        }
    });

    test("Go to definition on variable with redeclarations shows multiple", async () => {
        const { tmplUri } = await createDocument(
            "definition-redecl-test.tmpl",
            "{{ $test := 0 }}\n{{ $test }}\n{{ $test := 1 }}\n{{ $test }}",
        );

        try {
            await new Promise((resolve) => setTimeout(resolve, waitTime));

            // Execute definition provider on last $test usage (line 3, char 4)
            const definitions = await vscode.commands.executeCommand<
                vscode.Location[]
            >(
                "vscode.executeDefinitionProvider",
                tmplUri,
                new vscode.Position(3, 4),
            );

            assert.ok(definitions, "Definitions should be returned");
            assert.strictEqual(
                definitions.length,
                2,
                `Expected 2 definitions for redeclared variable, got ${definitions.length}`,
            );
        } finally {
            cleanupDocument(tmplUri);
        }
    });

    test("Go to definition on dot inside range points to range pipe", async () => {
        const { tmplUri } = await createDocument(
            "definition-dot-test.tmpl",
            "{{- range .Join }}\n{{ . }}\n{{- end }}",
        );

        try {
            await new Promise((resolve) => setTimeout(resolve, waitTime));

            // Execute definition provider on the dot (line 1, char 3)
            const definitions = await vscode.commands.executeCommand<
                vscode.Location[]
            >(
                "vscode.executeDefinitionProvider",
                tmplUri,
                new vscode.Position(1, 3),
            );

            assert.ok(definitions, "Definitions should be returned");
            assert.ok(
                definitions.length >= 1,
                `Expected at least 1 definition, got ${definitions.length}`,
            );
            assert.strictEqual(
                definitions[0].range.start.line,
                0,
                "Definition should point to range on line 0",
            );
        } finally {
            cleanupDocument(tmplUri);
        }
    });
});
