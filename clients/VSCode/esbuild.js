const esbuild = require("esbuild");
const fs = require("fs");
const path = require("path");

const production = process.argv.includes("--production");
const watch = process.argv.includes("--watch");

/**
 * @type {import('esbuild').Plugin}
 */
const copyBinariesPlugin = {
  name: "copy-binaries",
  setup(build) {
    build.onEnd(async (result) => {
      if (result.errors.length > 0) return;

      // Copy binaries from dist/server directory to dist/server/bin
      const srcBinDir = path.join(__dirname, "../../dist/server");
      const distBinDir = path.join(__dirname, "dist/server/bin");

      // Ensure dist/server/bin exists
      fs.mkdirSync(distBinDir, { recursive: true });

      const binaries = [
        "gotmpl-server",
        "gotmpl-server.exe",
        "gotmpl-server-arm64",
        "gotmpl-server-arm64.exe",
        "gotmpl-server-darwin-amd64",
        "gotmpl-server-darwin-arm64",
      ];
      binaries.forEach((binary) => {
        const src = path.join(srcBinDir, binary);
        if (fs.existsSync(src)) {
          const dst = path.join(distBinDir, binary);
          fs.copyFileSync(src, dst);
          // Make binary executable on Unix
          if (!binary.endsWith(".exe")) {
            fs.chmodSync(dst, 0o755);
          }
        }
      });
    });
  },
};

async function main() {
  const ctx = await esbuild.context({
    entryPoints: ["src/extension.ts"],
    bundle: true,
    format: "cjs",
    minify: production,
    sourcemap: !production,
    sourcesContent: false,
    platform: "node",
    outfile: "dist/extension.js",
    external: ["vscode"],
    logLevel: "warning",
    plugins: [
      copyBinariesPlugin,
      /* add to the end of plugins array */
      esbuildProblemMatcherPlugin,
    ],
  });
  if (watch) {
    await ctx.watch();
  } else {
    await ctx.rebuild();
    await ctx.dispose();
  }
}

/**
 * @type {import('esbuild').Plugin}
 */
const esbuildProblemMatcherPlugin = {
  name: "esbuild-problem-matcher",

  setup(build) {
    build.onStart(() => {
      console.log("[watch] build started");
    });
    build.onEnd((result) => {
      result.errors.forEach(({ text, location }) => {
        console.error(`[ERROR] ${text}`);
        if (location == null) return;
        console.error(
          `    ${location.file}:${location.line}:${location.column}:`,
        );
      });
      console.log("[watch] build finished");
    });
  },
};

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
