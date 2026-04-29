import * as path from 'path';
import { workspace, ExtensionContext } from 'vscode';

import {
    LanguageClient,
    LanguageClientOptions,
    ServerOptions,
    TransportKind
} from 'vscode-languageclient/node';

let client: LanguageClient | undefined;

export async function activate(context: ExtensionContext) {
    // gotmpl-server is the compiled Go binary, not ts or js, as the server is written in go
    const binaryName = process.platform === 'win32' ? 'gotmpl-server.exe' : 'gotmpl-server';
    let serverModule = context.asAbsolutePath(path.join('server', 'bin', binaryName));

    const serverOptions: ServerOptions = {
        module: serverModule,
        transport: TransportKind.stdio
    };

    const clientOptions: LanguageClientOptions = {
        documentSelector: [
            { scheme: "file", language: "gotmpl" }
        ],
        synchronize: {
            fileEvents: workspace.createFileSystemWatcher("**/*.*.tmpl")
        }
    };

    client = new LanguageClient(
        'gotmplLanguageServer',
        'Go Template Language Server',
        serverOptions,
        clientOptions
    );

    await client.start();
}

export async function deactivate() {
    if (!client) {
        return undefined
    }
    return client.stop();
}
