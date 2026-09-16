import { execSync } from 'child_process';
import { existsSync, mkdirSync } from 'fs';
import { dirname, join } from 'path';
import { fileURLToPath } from 'url';


console.log('Starting: Building server binaries');

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const serverDirectory = join(__dirname, '..', 'server');
const serverBinariesDirectory = join(__dirname, '..', 'dist', 'server')

if (!existsSync(serverBinariesDirectory)){
    mkdirSync(serverBinariesDirectory, {recursive: true});
}

// Platform configurations: [platform, arch, outputName]
const platforms = [
    ['windows', 'amd64', 'gotmpl-server.exe'],
    ['windows', 'arm64', 'gotmpl-server-arm64.exe'],
    ['darwin', 'amd64', 'gotmpl-server-darwin-amd64'],
    ['darwin', 'arm64', 'gotmpl-server-darwin-arm64'],
    ['linux', 'amd64', 'gotmpl-server'],
    ['linux', 'arm64', 'gotmpl-server-arm64'],
] as const;

console.log(`Building server binaries for ${platforms.length} platform(s)...`);

platforms.forEach(([goos, goarch, outputName]) => {
    const outputPath = join(serverBinariesDirectory, outputName);
    console.log(`  Building ${outputName} (GOOS=${goos} GOARCH=${goarch})...`);
    
    execSync(
        `go build -buildvcs=false -o "${outputPath}"`,
        {
            stdio: 'inherit',
            cwd: serverDirectory,
            env: {
                ...process.env,
                GOOS: goos,
                GOARCH: goarch,
                CGO_ENABLED: '0'
            }
        }
    );
});

// Freestanding, host-native binary. Unlike the cross-compiled targets above,
// this one is built for the machine running the build and is meant to be used
// directly as a standalone CLI/LSP executable. It is written to dist/ (not
// dist/server/) so it is not swept into the extension bundle by build:vscode.
const nativeOutputName = process.platform === 'win32' ? 'gotmpls.exe' : 'gotmpls';
const distDirectory = join(__dirname, '..', 'dist');
const nativeOutputPath = join(distDirectory, nativeOutputName);
console.log(`Building host-native binary ${nativeOutputName}...`);

execSync(
    `go build -buildvcs=false -o "${nativeOutputPath}"`,
    {
        stdio: 'inherit',
        cwd: serverDirectory,
        env: {
            ...process.env,
            CGO_ENABLED: '0'
        }
    }
);

// Lighter, check-only host-native binary. Built with the `cli` build tag, which
// excludes the LSP server runtime entirely; the binary runs the checker
// directly (no `check` subcommand to type).
const cliOutputName = process.platform === 'win32' ? 'gotmpls-check.exe' : 'gotmpls-check';
const cliOutputPath = join(distDirectory, cliOutputName);
console.log(`Building host-native check-only binary ${cliOutputName}...`);

execSync(
    `go build -buildvcs=false -tags cli -o "${cliOutputPath}"`,
    {
        stdio: 'inherit',
        cwd: serverDirectory,
        env: {
            ...process.env,
            CGO_ENABLED: '0'
        }
    }
);

console.log('All binaries built successfully');
