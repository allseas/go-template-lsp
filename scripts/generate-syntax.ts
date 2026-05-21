import { execSync } from "child_process";
import { copyFileSync } from "fs";
import { dirname, join } from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const root = join(__dirname, "..");
const syntaxesDir = join(root, "syntax", "syntaxes");

console.log("Generating syntax grammar...");
execSync("runghc Generate.hs", { stdio: "inherit", cwd: join(root, "syntax") });

// lint the generated syntax file
execSync(`npx prettier --write --config ${join(root, "clients", "VSCode", "package.json")} ${join(syntaxesDir, "gotemplate.tmLanguage.json")}`, {
  stdio: "inherit",
});

const src = join(syntaxesDir, "gotemplate.tmLanguage.json");
const destinations = [
  join(root, "clients", "VSCode", "syntaxes", "gotmpl.tmLanguage.json"),
  join(
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
    "gotmpl.tmLanguage.json",
  ),
];

for (const dest of destinations) {
  console.log(`  Copying to ${dest}`);
  copyFileSync(src, dest);
}

console.log("Done.");
