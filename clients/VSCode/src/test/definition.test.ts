import * as assert from "assert";
import * as fs from "fs";
import { after, before } from "mocha";
import * as os from "os";
import * as path from "path";
import * as vscode from "vscode";
import { cleanupDocument, createDocument } from "./utils";

const testResourcesDir = path.join(__dirname, "../../test/resources");
const definitionTestsSourceDir = path.join(
    __dirname,
    "../../../../test/resources/definition-tests-client",
);

const serverDebugLog = path.join(os.tmpdir(), "gotmpl-server-debug.log");
// Set before the extension activates so the spawned server inherits it.
process.env.GOTMPL_DEBUG_LOG = serverDebugLog;
try {
    fs.unlinkSync(serverDebugLog);
} catch {
    // ignore — file may not exist
}

function readServerDebugLog(): string {
    try {
        return fs.readFileSync(serverDebugLog, "utf8");
    } catch {
        return "(no server debug log produced)";
    }
}

suite("Definition Test Suite", () => {
    before(() => {
        fs.mkdirSync(path.join(testResourcesDir, "model"), { recursive: true });
        fs.copyFileSync(
            path.join(definitionTestsSourceDir, "go.mod"),
            path.join(testResourcesDir, "go.mod"),
        );
        fs.copyFileSync(
            path.join(definitionTestsSourceDir, "model", "model.go"),
            path.join(testResourcesDir, "model", "model.go"),
        );
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
            const definitions = await getDefinitions(
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
            // Execute definition provider on last $test usage (line 3, char 4)
            const definitions = await getDefinitions(
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
            const definitions = await getDefinitions(
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

    test("Go to definition on field jumps to struct field declaration", async () => {
        const { tmplUri } = await createDocument(
            "definition-field-test.tmpl",
            "{{/*gotype: cg/model.Order*/}}\n{{ .CustomerName }}",
        );

        try {
            assert.ok(
                fs.existsSync(path.join(testResourcesDir, "go.mod")),
                "go.mod file should exist",
            );
            assert.ok(
                fs.existsSync(path.join(testResourcesDir, "model")),
                "model directory should exist",
            );

            // char 5 is inside "CustomerName" (after "{{ .")
            const definitions = await pollDefinitions(
                tmplUri,
                new vscode.Position(1, 5),
            );

            assert.ok(definitions, "Definitions should be returned");
            assert.ok(
                definitions.length >= 1,
                `Expected at least 1 definition, got ${definitions.length}. Server log:\n${readServerDebugLog()}`,
            );
            assert.ok(
                definitions[0].uri.fsPath.endsWith("model.go"),
                `Expected definition in model.go, got ${definitions[0].uri.fsPath}`,
            );
            assert.strictEqual(
                definitions[0].range.start.line,
                70,
                "CustomerName should be on line 71 (0-indexed: 70)",
            );
        } finally {
            cleanupDocument(tmplUri);
        }
    });

    test("Go to definition on method jumps to method declaration", async () => {
        const { tmplUri } = await createDocument(
            "definition-method-test.tmpl",
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
                `Expected at least 1 definition, got ${definitions.length}. Server log:\n${readServerDebugLog()}`,
            );
            assert.ok(
                definitions[0].uri.fsPath.endsWith("model.go"),
                `Expected definition in model.go, got ${definitions[0].uri.fsPath}`,
            );
            assert.strictEqual(
                definitions[0].range.start.line,
                79,
                "DisplayName should be on line 80 (0-indexed: 79)",
            );
        } finally {
            cleanupDocument(tmplUri);
        }
    });

    test("Go to definition on nested field first identifier jumps to field", async () => {
        const { tmplUri } = await createDocument(
            "definition-nested-first-test.tmpl",
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
                `Expected at least 1 definition, got ${definitions.length}. Server log:\n${readServerDebugLog()}`,
            );
            assert.ok(
                definitions[0].uri.fsPath.endsWith("model.go"),
                `Expected definition in model.go, got ${definitions[0].uri.fsPath}`,
            );
            assert.strictEqual(
                definitions[0].range.start.line,
                72,
                "Address field should be on line 73 (0-indexed: 72)",
            );
        } finally {
            cleanupDocument(tmplUri);
        }
    });

    test("Go to definition on nested field second identifier jumps to nested field", async () => {
        const { tmplUri } = await createDocument(
            "definition-nested-second-test.tmpl",
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
                `Expected at least 1 definition, got ${definitions.length}. Server log:\n${readServerDebugLog()}`,
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
            cleanupDocument(tmplUri);
        }
    });

    test("Go to definition on unknown field returns no results", async () => {
        const { tmplUri } = await createDocument(
            "definition-unknown-field-test.tmpl",
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
            cleanupDocument(tmplUri);
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
