import * as assert from "assert";
import { after, before } from "mocha";
import * as path from "path";
import * as vscode from "vscode";

suite("Definition Test Suite", () => {
    before(async () => {
        await new Promise((resolve) => setTimeout(resolve, 2000));
    });

    after(() => {
        vscode.window.showInformationMessage("All definition tests done!");
    });

    test("Go to definition on variable usage jumps to declaration", async () => {
        const tmplUri = vscode.Uri.file(
            path.join(__dirname, "../../test/resources/definition-test.tmpl"),
        );
        const edit = new vscode.WorkspaceEdit();
        edit.createFile(tmplUri, { overwrite: true });
        edit.insert(
            tmplUri,
            new vscode.Position(0, 0),
            "{{ $test := 0 }}\n{{ $test }}",
        );
        await vscode.workspace.applyEdit(edit);

        const document = await vscode.workspace.openTextDocument(tmplUri);
        await vscode.window.showTextDocument(document);

        try {
            // Wait for the language server to be ready
            await new Promise((resolve) => setTimeout(resolve, 3000));

            // Execute definition provider on $test usage (line 1, char 4)
            const definitions =
                await vscode.commands.executeCommand<vscode.Location[]>(
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
            await vscode.commands.executeCommand(
                "workbench.action.closeActiveEditor",
            );
            const deleteEdit = new vscode.WorkspaceEdit();
            deleteEdit.deleteFile(tmplUri);
            await vscode.workspace.applyEdit(deleteEdit);
        }
    });

    test("Go to definition on variable with redeclarations shows multiple", async () => {
        const tmplUri = vscode.Uri.file(
            path.join(
                __dirname,
                "../../test/resources/definition-redecl-test.tmpl",
            ),
        );
        const edit = new vscode.WorkspaceEdit();
        edit.createFile(tmplUri, { overwrite: true });
        edit.insert(
            tmplUri,
            new vscode.Position(0, 0),
            "{{ $test := 0 }}\n{{ $test }}\n{{ $test := 1 }}\n{{ $test }}",
        );
        await vscode.workspace.applyEdit(edit);

        const document = await vscode.workspace.openTextDocument(tmplUri);
        await vscode.window.showTextDocument(document);

        try {
            await new Promise((resolve) => setTimeout(resolve, 3000));

            // Execute definition provider on last $test usage (line 3, char 4)
            const definitions =
                await vscode.commands.executeCommand<vscode.Location[]>(
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
            await vscode.commands.executeCommand(
                "workbench.action.closeActiveEditor",
            );
            const deleteEdit = new vscode.WorkspaceEdit();
            deleteEdit.deleteFile(tmplUri);
            await vscode.workspace.applyEdit(deleteEdit);
        }
    });

    test("Go to definition on dot inside range points to range pipe", async () => {
        const tmplUri = vscode.Uri.file(
            path.join(
                __dirname,
                "../../test/resources/definition-dot-test.tmpl",
            ),
        );
        const edit = new vscode.WorkspaceEdit();
        edit.createFile(tmplUri, { overwrite: true });
        edit.insert(
            tmplUri,
            new vscode.Position(0, 0),
            "{{- range .Join }}\n{{ . }}\n{{- end }}",
        );
        await vscode.workspace.applyEdit(edit);

        const document = await vscode.workspace.openTextDocument(tmplUri);
        await vscode.window.showTextDocument(document);

        try {
            await new Promise((resolve) => setTimeout(resolve, 3000));

            // Execute definition provider on the dot (line 1, char 3)
            const definitions =
                await vscode.commands.executeCommand<vscode.Location[]>(
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
            await vscode.commands.executeCommand(
                "workbench.action.closeActiveEditor",
            );
            const deleteEdit = new vscode.WorkspaceEdit();
            deleteEdit.deleteFile(tmplUri);
            await vscode.workspace.applyEdit(deleteEdit);
        }
    });
});
