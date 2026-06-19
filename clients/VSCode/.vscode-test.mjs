// .vscode-test.js
import { defineConfig } from "@vscode/test-cli";

const baseConfig = {
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
};

export default defineConfig([
    {
        label: "default",
        ...baseConfig,
    },
    {
        label: "allseas",
        ...baseConfig,
    },
    {
        label: "benchmark",
        ...baseConfig,
        files: "out/test/benchmark.test.js",
        mocha: {
            ...baseConfig.mocha,
            timeout: 300_000,
        },
    },
]);
