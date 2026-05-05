import * as path from "path";
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

  const isDebug =
    process.env.VSCODE_DEBUG === "true" ||
    process.env.NODE_ENV !== "production";
  const buildFolder = isDebug ? "out" : "dist";
  const serverModule = context.asAbsolutePath(
    path.join(buildFolder, "server", "bin", binaryName),
  );
  console.log("Extension build folder:", buildFolder);

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
    documentSelector: [{ scheme: "file", language: "gotmpl" }],
    synchronize: {
      fileEvents: watchers,
      configurationSection: "goTmplSupport",
    },
  };

  client = new LanguageClient(
    "gotmplLanguageServer",
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
}

export async function deactivate() {
  if (!client) {
    return undefined;
  }
  return client.stop();
}
