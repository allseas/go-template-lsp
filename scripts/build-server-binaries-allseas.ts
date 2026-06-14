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
        `go build -o "${outputPath}" -tags "allseas"`,
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

console.log('All binaries built successfully');
