// .vscode-test.js
import { defineConfig } from "@vscode/test-cli";

export default defineConfig([
    {
        label: "unitTests",
        files: "out/test/**/*.test.js",
        version: "stable",
        workspaceFolder: "../../test/resources/templ-tests",
        extensionDevelopmentPath: "./",
        launchArgs: ["--extensionDevelopmentPath=."],
        mocha: {
            ui: "tdd",
            timeout: 20000,
        },
    },
    // you can specify additional test configurations, too
]);
