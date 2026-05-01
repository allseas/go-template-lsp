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

#### Running the extension with watching

1. Run the watcher for server and extension source code:
```
npm run watch:vscode
```

2. Open VS Code in the `clients/VSCode` folder. Then press `F5` to run a new VS Code window with the extension.

3. For logs from the server, look into `Output` in the new VS Code window.

#### Running the extension with static builds

1. Build the binaries and compile the extension:
```
npm run build:vscode
```

2. You also need to manually copy the server binaries from `/dist/server` to the `out` folder where the extension is compiled. 

3. Open VS Code in the `clients/VSCode` folder. Then press `F5` to run a new VS Code window with the extension.

4. For logs from the server, look into `Output` in the new VS Code window.

#### Packaging the extension

1. Build the binaries and package the extension:
```
npm run package:vscode
```

2. Open VS Code in the `clients/VSCode` folder. Then press `F5` to run a new VS Code window with the extension.

3. For logs from the server, look into `Output` in the new VS Code window.
