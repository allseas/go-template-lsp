# Go Text Template Support

You are running version **0.1.0**!

Features currently supported:

- Syntax highlighting.
- Auto completions - template variables and built-in functions. The suggestions are scope aware.
- Find references/usages of a function or variable. The default shortcut is `Shift+Alt+F12`.
- Wrap selection in a comment. The default shortcut is `Ctrl+/`.
- Wrap selection in a tag block - done using snippets. In order to wrap in an if you block you need to select the text you want to surround, type `wif` and press tab. You can see other wrap snippets in the `Ctrl+Space` menu.
- Snippets for tags. You can see them by pressing `Ctrl+Space`.
- User/project configuration. You can edit it in the Settings UI (`Ctrl+Space`) in the `Extensions` category under `text/template Support`.
- Hover definitons - when hovering over a function or variable.

If you encouter any bugs please messages us with the description and server logs. You can find them in VS Code in the `Output` tab next to the terminal when selecting `Go Template Language Server`.
