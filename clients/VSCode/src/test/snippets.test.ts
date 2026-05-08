//https://github.com/microsoft/vscode-extension-samples/blob/main/helloworld-test-sample/src/test/suite/extension.test.ts
import * as assert from "assert";
import { after, before } from "mocha";
import * as path from "path";

import * as vscode from "vscode";
// import * as myExtension from '../extension';

suite("Snippets Test Suite", () => {
    before(async () => {
        await new Promise(resolve => setTimeout(resolve, 1000));
    });

    after(() => {
        vscode.window.showInformationMessage("All snippet tests done!");
    });

    test("Snippets should be available in .tmpl files", async () => {
        // Create a temporary .tmpl file
        const tmplUri = vscode.Uri.file(path.join(__dirname, "../../test/resources/snippets-test.tmpl"));
        const edit = new vscode.WorkspaceEdit();
        edit.createFile(tmplUri, { overwrite: true });
        edit.insert(tmplUri, new vscode.Position(0, 0), "if");
        await vscode.workspace.applyEdit(edit);

        const document = await vscode.workspace.openTextDocument(tmplUri);
        const editor = await vscode.window.showTextDocument(document);

        try {
            assert.strictEqual(
                document.languageId,
                "gotmpl",
                "File should be recognized as gotmpl language"
            );

            const result = await vscode.commands.executeCommand(
                "vscode.executeCompletionItemProvider",
                tmplUri,
                new vscode.Position(0, 0)
            );


            const snippets = (result as any)?.items || [];
            assert.ok(Array.isArray(snippets), "CompletionList should have items array");


            snippets.forEach((item: any, index: number) => {
                const label = typeof item.label === 'string' ? item.label : item.label?.label;
                console.log(`  ${index}: ${label}`);
            });

            const templateSnippets = (snippets as any[]).filter((item: any) => {
                const label = typeof item.label === 'string' ? item.label : item.label?.label;
                return ["if", "range", "with", "break", "continue", "else", "var", "template"].includes(label as string);
            });

            assert.ok(
                templateSnippets.length > 0,
                `Template snippets should be available in .tmpl files. Got: ${snippets.length} completions, snippet labels: ${snippets.slice(0, 5).map((s: any) => typeof s.label === 'string' ? s.label : s.label?.label).join(", ")}`
            );
        } finally {
            await vscode.commands.executeCommand("workbench.action.closeActiveEditor");
            const deleteEdit = new vscode.WorkspaceEdit();
            deleteEdit.deleteFile(tmplUri);
            await vscode.workspace.applyEdit(deleteEdit);
        }
    });

    test("Snippets should NOT be available in non-.tmpl files", async () => {
        // Create a temporary .txt file
        const txtUri = vscode.Uri.file(path.join(__dirname, "../../test/resources/snippets-test.txt"));
        const edit = new vscode.WorkspaceEdit();
        edit.createFile(txtUri, { overwrite: true });
        edit.insert(txtUri, new vscode.Position(0, 0), "if");
        await vscode.workspace.applyEdit(edit);

        const document = await vscode.workspace.openTextDocument(txtUri);
        const editor = await vscode.window.showTextDocument(document);

        try {
            assert.notStrictEqual(
                document.languageId,
                "gotmpl",
                "File should NOT be recognized as gotmpl language"
            );

            const result = await vscode.commands.executeCommand(
                "vscode.executeCompletionItemProvider",
                txtUri,
                new vscode.Position(0, 0)
            );

            const completions = (result as any)?.items || [];
            assert.ok(Array.isArray(completions), "CompletionList should have items array");

            const templateSnippets = (completions as any[]).filter((item: any) => {
                const label = typeof item.label === 'string' ? item.label : item.label?.label;
                return ["if", "range", "with", "break", "continue", "else", "var", "template"].includes(label as string);
            });

            assert.strictEqual(
                templateSnippets.length,
                0,
                `Template snippets should NOT be available in non-.tmpl files. Got ${templateSnippets.length} snippets: ${templateSnippets.map((s: any) => typeof s.label === 'string' ? s.label : s.label?.label).join(", ")}`
            );
        } finally {
            await vscode.commands.executeCommand("workbench.action.closeActiveEditor");
            const deleteEdit = new vscode.WorkspaceEdit();
            deleteEdit.deleteFile(txtUri);
            await vscode.workspace.applyEdit(deleteEdit);
        }
    });
});
