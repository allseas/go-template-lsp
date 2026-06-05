# Go Text Template Support

You are running version **0.1.0**!

Features currently supported:

- Syntax highlighting.
- Auto completions - template variables, built-in functions, and chained field accesses. The suggestions are scope aware.
- Go to Definition - navigate to variable and function definitions with `Ctrl+Click` or `F12`.
- Find references/usages of a function or variable. The default shortcut is `Shift+Alt+F12`.
- Inspections - detection of incorrect syntax and duplicate variable names.
- Wrap selection in a comment. The default shortcut is `Ctrl+/`.
- Wrap selection in a tag block - done using snippets. In order to wrap in an if you block you need to select the text you want to surround, type `wif` and press tab. You can see other wrap snippets in the `Ctrl+Space` menu.
- Snippets for tags. You can see them by pressing `Ctrl+Space`.
- User/project configuration. You can edit it in the Settings UI (`Ctrl+Space`) in the `Extensions` category under `text/template Support`, or save configuration in a `gotmpl.config.json` file in your project.
- Hover definitions - when hovering over a function or variable.

If you encounter any bugs please message us with the description and server logs. You can find them in VS Code in the `Output` tab next to the terminal when selecting `Go Template Language Server`.
