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
// Oniguruma is required by vscode-textmate to execute TextMate grammar rules; VSCode uses the same regex engine internally.
let _grammar: vsctm.IGrammar | null = null;
let _wasmLoaded = false;

async function ensureOniguruma() {
    if (_wasmLoaded) return;
    const wasmBin = fs.readFileSync(
        path.join(
            __dirname,
            "../../../../node_modules/vscode-oniguruma/release/onig.wasm",
        ),
    );
    await oniguruma.loadWASM(wasmBin.buffer as ArrayBuffer);
    _wasmLoaded = true;
}

function localGrammarPath(fileName: string): string {
    return path.join(__dirname, "../../syntaxes/" + fileName);
}

// stubGrammar creates a minimal grammar whose only rule tags every character
// with a marker scope. It lets us verify that an outer grammar's
// `include: source.X` reference correctly hands off tokenization to `source.X`
// without needing VSCode's built-in language grammars to be loaded.
function stubGrammar(scopeName: string): vsctm.IRawGrammar {
    return vsctm.parseRawGrammar(
        JSON.stringify({
            scopeName,
            patterns: [{ match: ".", name: `${scopeName}.stub.marker` }],
        }),
        `${scopeName}.stub.json`,
    );
}

export async function getGrammar(): Promise<vsctm.IGrammar> {
    if (_grammar) return _grammar;
    await ensureOniguruma();
    const registry = new vsctm.Registry({
        onigLib: Promise.resolve({
            createOnigScanner: (patterns: string[]) =>
                new oniguruma.OnigScanner(patterns),
            createOnigString: (s: string) => new oniguruma.OnigString(s),
        }),
        loadGrammar: async (scopeName: string) => {
            if (scopeName === "source.gotmpl") {
                const grammarPath = localGrammarPath("gotmpl.tmLanguage.json");
                return vsctm.parseRawGrammar(
                    fs.readFileSync(grammarPath, "utf-8"),
                    grammarPath,
                );
            }
            return null;
        },
    });
    const grammar = await registry.loadGrammar("source.gotmpl");
    if (!grammar) throw new Error("Failed to load gotmpl grammar");
    _grammar = grammar;
    return grammar;
}

// getEmbeddedGrammar loads a gotmpl-<lang> grammar (e.g. "gotmpl-cpp") together
// with the base gotmpl grammar and a stub for the embedded language scope
// (e.g. "source.cpp"). Returns a tokenizer for the wrapper grammar.
export async function getEmbeddedGrammar(
    variant: string,
    embeddedScope: string,
): Promise<vsctm.IGrammar> {
    await ensureOniguruma();
    const wrapperScope = `source.gotmpl.${variant}`;
    const wrapperPath = localGrammarPath(
        `gotmpl-${variant}.tmLanguage.json`,
    );
    const gotmplPath = localGrammarPath("gotmpl.tmLanguage.json");

    const registry = new vsctm.Registry({
        onigLib: Promise.resolve({
            createOnigScanner: (patterns: string[]) =>
                new oniguruma.OnigScanner(patterns),
            createOnigString: (s: string) => new oniguruma.OnigString(s),
        }),
        loadGrammar: async (scopeName: string) => {
            if (scopeName === wrapperScope) {
                return vsctm.parseRawGrammar(
                    fs.readFileSync(wrapperPath, "utf-8"),
                    wrapperPath,
                );
            }
            if (scopeName === "source.gotmpl") {
                return vsctm.parseRawGrammar(
                    fs.readFileSync(gotmplPath, "utf-8"),
                    gotmplPath,
                );
            }
            if (scopeName === embeddedScope) {
                return stubGrammar(embeddedScope);
            }
            return null;
        },
    });

    const grammar = await registry.loadGrammar(wrapperScope);
    if (!grammar) throw new Error(`Failed to load ${wrapperScope} grammar`);
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
