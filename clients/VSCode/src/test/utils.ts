import * as vscode from "vscode";
import * as path from "path";
import { Uri } from "vscode";

export async function createDocument(fileName: string, fileContents: string) {
    const tmplUri = vscode.Uri.file(
        path.join(__dirname, "../../test/resources/" + fileName),
    );
    const edit = new vscode.WorkspaceEdit();
    edit.createFile(tmplUri, { overwrite: true });
    edit.insert(tmplUri, new vscode.Position(0, 0), fileContents);
    await vscode.workspace.applyEdit(edit);

    const document = await vscode.workspace.openTextDocument(tmplUri);
    await vscode.window.showTextDocument(document);
    return { tmplUri, document };
}

export async function cleanupDocument(fileUri: Uri) {
    await vscode.commands.executeCommand("workbench.action.closeActiveEditor");
    const deleteEdit = new vscode.WorkspaceEdit();
    deleteEdit.deleteFile(fileUri);
    await vscode.workspace.applyEdit(deleteEdit);
}
