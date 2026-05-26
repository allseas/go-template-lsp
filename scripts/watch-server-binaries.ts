import { execSync } from "child_process";
import { dirname, join } from "path";
import { fileURLToPath } from "url";

let binaryName: string;

if (process.platform === 'win32') {
    binaryName = process.arch === 'arm64' ? 'gotmpl-server-arm64.exe' : 'gotmpl-server.exe';
} else if (process.platform === 'darwin') { // macOS
    binaryName = process.arch === 'arm64' ? 'gotmpl-server-darwin-arm64' : 'gotmpl-server-darwin-amd64';
} else {
    binaryName = process.arch === 'arm64' ? 'gotmpl-server-arm64' : 'gotmpl-server';
}

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const serverDirectory = join(__dirname, '..', 'server');

const binariesDir = join(__dirname, '..', 'clients', 'VSCode', 'dist', 'server', 'bin');

const outputPath = join(binariesDir, binaryName);

execSync(
    `gowatch -o "${outputPath}"`,
    {
        stdio: 'inherit',
        cwd: serverDirectory,
        env: {
            ...process.env,
            CGO_ENABLED: '0'
        }
    }
);
