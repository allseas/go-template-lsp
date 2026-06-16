// .vscode-test.js
import { defineConfig } from "@vscode/test-cli";

export default defineConfig([
    {
        label: "unitTests",
        files: "out/test/**/*.test.js",
        version: "stable",
        workspaceFolder: "../../test/resources/templ-tests",
        extensionDevelopmentPath: "./",
        launchArgs: [
            "--extensionDevelopmentPath=.",
            "--user-data-dir=/tmp/vscode-test-userdata",
            "--no-sandbox",
            "--disable-gpu",
            "--disable-dev-shm-usage",
        ],
        mocha: {
            ui: "tdd",
            timeout: 20000,
        },
    },
    // you can specify additional test configurations, too
]);
