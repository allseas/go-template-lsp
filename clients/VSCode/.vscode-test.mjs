// .vscode-test.js
import { defineConfig } from "@vscode/test-cli";

export default defineConfig([
    {
        label: "unitTests",
        files: "out/test/**/*.test.js",
        version: "insiders",
        workspaceFolder: "../../test/resources/templ-tests",
        mocha: {
            ui: "tdd",
            timeout: 20000,
        },
    },
    // you can specify additional test configurations, too
]);
