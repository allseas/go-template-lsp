# GoTemplate Support Extension

## Development

### Requirements

- Go: 1.26.2+
- Node: 24.11.0+
- Npm: 11.12.0+
- gowatch (https://github.com/silenceper/gowatch)

### VS Code extension

#### Prerequisites:

Install the npm packages:
```
npm i
cd clients/VSCode
npm i
cd ../..
```

#### Process

Run the watcher for server and extension source code:
```
npm run watch:vscode
```

Open VS Code in the `clients/VSCode` folder. Then press `F5` to run a new VS Code window with the extension.

For logs from the server, look into `Output` in the new VS Code window.
