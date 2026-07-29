# Go Text Template Support

You are running version **1.2.0**!

- Semantic syntax highlighting
- Auto completions (variables, built-in functions, user-defined functions, chained field accesses)
- Go to Definition (`Ctrl+Click` or `F12`) — including `{{ template "name" }}` calls
- Find references (`Shift+Alt+F12`)
- Refactor rename (`F2`) for variables and functions
- Hover information — user-defined functions show their godoc-style comment and input/output types
- Diagnostics (syntax errors, duplicate variables, type errors, etc.)
- Type checking on `{{ template }}` blocks
- `map[string]any` type hints with completions, hover, go-to-definition and diagnostics
- Type hints can be placed anywhere in the template
- Wrap selection in a comment (`Ctrl+/`)
- Wrap selection in a tag block (via snippets)
- Snippets for tags
- User/project configuration (`gotmpl.config.json`), including a custom language server binary path

If you encounter any bugs please message us with the description and server logs. You can find them in VS Code in the `Output` tab next to the terminal when selecting `Go Template Language Server`.
