import { execSync } from "child_process";
import { copyFileSync, readdirSync } from "fs";
import { dirname, join } from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const root = join(__dirname, "..");
const syntaxesDir = join(root, "syntax", "syntaxes");

const vscodeSyntaxesDir = join(root, "clients", "VSCode", "syntaxes");
const jetbrainsSyntaxesDir = join(
  root,
  "clients",
  "JetBrains",
  "go-text-template",
  "src",
  "main",
  "resources",
  "textmate",
  "go-text-template",
  "Syntaxes",
);

console.log("Generating syntax grammar...");
execSync("cabal run", { stdio: "inherit", cwd: join(root, "syntax") });

const generated = readdirSync(syntaxesDir).filter((f) =>
  f.endsWith(".tmLanguage.json"),
);

// lint every generated file
for (const file of generated) {
  execSync(
    `npx prettier --write --config ${join(root, "clients", "VSCode", "package.json")} ${join(syntaxesDir, file)}`,
    { stdio: "inherit" },
  );
}

// gotemplate.tmLanguage.json       -> gotmpl.tmLanguage.json       (base)
// gotemplate-<key>.tmLanguage.json -> gotmpl-<key>.tmLanguage.json (derived)
function destName(file: string): string {
  if (file === "gotemplate.tmLanguage.json") return "gotmpl.tmLanguage.json";
  const m = file.match(/^gotemplate-(.+)\.tmLanguage\.json$/);
  if (!m) throw new Error(`Unexpected grammar file name: ${file}`);
  return `gotmpl-${m[1]}.tmLanguage.json`;
}

for (const file of generated) {
  const src = join(syntaxesDir, file);
  const outName = destName(file);
  for (const dir of [vscodeSyntaxesDir, jetbrainsSyntaxesDir]) {
    const dest = join(dir, outName);
    console.log(`  Copying to ${dest}`);
    copyFileSync(src, dest);
  }
}

console.log("Done.");
