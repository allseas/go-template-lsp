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
    let binaryName: string;
    
    if (process.platform === 'win32') {
        binaryName = process.arch === 'arm64' ? 'gotmpl-server-arm64.exe' : 'gotmpl-server.exe';
    } else if (process.platform === 'darwin') { // macOS
        binaryName = process.arch === 'arm64' ? 'gotmpl-server-darwin-arm64' : 'gotmpl-server-darwin-amd64';
    } else {
        binaryName = process.arch === 'arm64' ? 'gotmpl-server-arm64' : 'gotmpl-server';
    }
    
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
