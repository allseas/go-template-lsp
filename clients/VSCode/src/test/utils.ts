import * as vscode from "vscode";
import * as path from "path";
import { Uri } from "vscode";
import * as vsctm from "vscode-textmate";
import * as oniguruma from "vscode-oniguruma";
import * as fs from "fs";

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

let _grammar: vsctm.IGrammar | null = null;

export async function getGrammar(): Promise<vsctm.IGrammar> {
    if (_grammar) return _grammar;

    const wasmBin = fs.readFileSync(
        path.join(
            __dirname,
            "../../../../node_modules/vscode-oniguruma/release/onig.wasm",
        ),
    );
    await oniguruma.loadWASM(wasmBin.buffer as ArrayBuffer);

    const registry = new vsctm.Registry({
        onigLib: Promise.resolve({
            createOnigScanner: (patterns: string[]) =>
                new oniguruma.OnigScanner(patterns),
            createOnigString: (s: string) => new oniguruma.OnigString(s),
        }),
        loadGrammar: async (scopeName: string) => {
            if (scopeName === "source.gotmpl") {
                const grammarPath = path.join(
                    __dirname,
                    "../../syntaxes/gotmpl.tmLanguage.json",
                );
                const content = fs.readFileSync(grammarPath, "utf-8");
                return vsctm.parseRawGrammar(content, grammarPath);
            }
            return null;
        },
    });

    const grammar = await registry.loadGrammar("source.gotmpl");
    if (!grammar) throw new Error("Failed to load gotmpl grammar");
    _grammar = grammar;
    return grammar;
}

export function getScopes(
    grammar: vsctm.IGrammar,
    line: string,
    character: number,
): string[] {
    const result = grammar.tokenizeLine(line, vsctm.INITIAL);
    for (const token of result.tokens) {
        if (token.startIndex <= character && character < token.endIndex) {
            return token.scopes;
        }
    }
    return [];
}

export function assertScope(scopes: string[], expectedScope: string) {
    if (!scopes.includes(expectedScope)) {
        throw new Error(
            `Expected scope "${expectedScope}", got: [${scopes.join(", ")}]`,
        );
    }
}
