import * as fs from "fs";
import * as path from "path";
import * as vscode from "vscode";
import {
    workspace,
    ExtensionContext,
    RelativePattern,
    FileSystemWatcher,
} from "vscode"; //Uri removed due to linter

import {
    LanguageClient,
    LanguageClientOptions,
    ServerOptions,
    TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

export async function activate(context: ExtensionContext) {
    console.log("Extension activated!");

    let binaryName: string;

    if (process.platform === "win32") {
        binaryName =
            process.arch === "arm64"
                ? "gotmpl-server-arm64.exe"
                : "gotmpl-server.exe";
    } else if (process.platform === "darwin") {
        // macOS
        binaryName =
            process.arch === "arm64"
                ? "gotmpl-server-darwin-arm64"
                : "gotmpl-server-darwin-amd64";
    } else {
        binaryName =
            process.arch === "arm64" ? "gotmpl-server-arm64" : "gotmpl-server";
    }

    const distPath = context.asAbsolutePath(
        path.join("dist", "server", "bin", binaryName),
    );
    const outPath = context.asAbsolutePath(
        path.join("out", "server", "bin", binaryName),
    );
    const serverModule = fs.existsSync(distPath) ? distPath : outPath;

    const serverOptions: ServerOptions = {
        command: serverModule,
        transport: TransportKind.stdio,
    };

    const folders = workspace.workspaceFolders;

    const watchers: FileSystemWatcher[] = [];

    if (!folders || folders.length === 0) {
        console.log("No workspace folder is open");
        return;
    }

    for (const folder of folders) {
        console.log(`Watching workspace: ${folder.uri.fsPath}`);

        const watcher = workspace.createFileSystemWatcher(
            new RelativePattern(folder, "**/*.*.tmpl"),
        );

        watchers.push(watcher);

        context.subscriptions.push(
            watcher,
            watcher.onDidCreate((uri) => console.log(`Created: ${uri.fsPath}`)),
            watcher.onDidChange((uri) => console.log(`Changed: ${uri.fsPath}`)),
            watcher.onDidDelete((uri) => console.log(`Deleted: ${uri.fsPath}`)),
        );
    }

    console.log("Server binary:", serverModule);

    const clientOptions: LanguageClientOptions = {
        documentSelector: [
            { scheme: "file", language: "gotmpl" },
            { scheme: "file", language: "gotmpl-sql" },
            { scheme: "file", language: "gotmpl-html" },
            { scheme: "file", language: "gotmpl-json" },
            { scheme: "file", language: "gotmpl-yaml" },
            { scheme: "file", language: "gotmpl-css" },
            { scheme: "file", language: "gotmpl-js" },
            { scheme: "file", language: "gotmpl-xml" },
            { scheme: "file", language: "gotmpl-md" },
            { scheme: "file", language: "gotmpl-sh" },
            { scheme: "file", language: "gotmpl-scl" },
            { scheme: "file", language: "gotmpl-cpp" },
        ],
        synchronize: {
            fileEvents: watchers,
            configurationSection: "goTmplSupport",
        },
    };

    client = new LanguageClient(
        "goTmplSupport",
        "Go Template Language Server",
        serverOptions,
        clientOptions,
    );

    try {
        await client.start();
        console.log("Language client started");
    } catch (err) {
        console.error("Language client failed:", err);
    }

    const versionKey = "goTmplSupport.version";

    const previousVersion = context.globalState.get<string>(versionKey);
    const currentVersion = context.extension.packageJSON.version;

    if (previousVersion === undefined || previousVersion !== currentVersion) {
        const welcomeFilePath = vscode.Uri.file(
            path.join(context.extensionPath, "welcome.md"),
        );

        vscode.commands.executeCommand("markdown.showPreview", welcomeFilePath);
        context.globalState.update(versionKey, currentVersion);
    }
}

export async function deactivate() {
    if (!client) {
        return undefined;
    }
    return client.stop();
}
