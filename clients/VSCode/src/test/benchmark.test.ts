/**
 * LSP performance benchmark - VS Code integration test
 *
 * Verifies the three timing requirements against a ~5 000-line template file.
 *
 *   1. Dynamic analysis (completions)
 *        average response ≤ 1 000 ms
 *        95th percentile  ≤ 2 000 ms
 *   2. Jump to definition   mean ≤ 1 000 ms
 *   3. Find usages          mean ≤ 1 000 ms
 *
 * Run via:  npm run benchmark  (or npm run benchmark:allseas)
 */

import * as assert from "assert";
import { after, before } from "mocha";
import * as vscode from "vscode";
import { cleanupDocument, createDocument } from "./utils";

// ---- Thresholds ----------------------------------------------------------

const THRESHOLD_AVERAGE_MS = 1_000;
const THRESHOLD_P95_MS = 2_000;
const THRESHOLD_DEFINITION_MS = 1_000;
const THRESHOLD_REFERENCES_MS = 1_000;

// ---- Iterations ----------------------------------------------------------

const WARMUP_ITERATIONS = 3;
const BENCH_ITERATIONS = 20;

// ---- Fixture positions (0-indexed line / character) ----------------------
//
// The generated fixture starts with these exact lines:
//
//  0  {{- /*gotype: cg/model.Model*/ -}}
//  1  {{ define "benchmark.main" }}
//  2  {{- /*gotype: cg/model.Instance*/ -}}
//  3  {{- $counter := 0 }}
//  4  {{- $name := .Name -}}
//  5  {{- template "benchmark.helper" . }}
//  6  Count: {{ $counter }}
//  7  Name: {{ $name }}
//  8  {{- $counter = add $counter 1 }}
//  9  {{- $counter = add $counter 1 }}
// 10  {{- $counter = add $counter 1 }}
// 11  {{- end }}
// 12
// 13  {{ define "benchmark.helper" }}
// ...

/** Inside `"benchmark.helper"` on the template-call line (for definition). */
const DEF_POS = new vscode.Position(5, 15); // 'b' of "benchmark.helper"

/** On the `$counter` variable declaration (for references). */
const REF_POS = new vscode.Position(3, 5); // 'c' of $counter

/** After the dot in `.Name` on line 4 (for completions). */
const COMPL_POS = new vscode.Position(4, 14); // 'N' of .Name

// ---- Fixture generation --------------------------------------------------

function generateFixture(targetLines: number): string {
    const header = [
        "{{- /*gotype: cg/model.Model*/ -}}",
        '{{ define "benchmark.main" }}',
        "{{- /*gotype: cg/model.Instance*/ -}}",
        "{{- $counter := 0 }}",
        "{{- $name := .Name -}}",
        '{{- template "benchmark.helper" . }}',
        "Count: {{ $counter }}",
        "Name: {{ $name }}",
        "{{- $counter = add $counter 1 }}",
        "{{- $counter = add $counter 1 }}",
        "{{- $counter = add $counter 1 }}",
        "{{- end }}",
        "",
        '{{ define "benchmark.helper" }}',
        "{{- /*gotype: cg/model.Instance*/ -}}",
        "This is the helper template.",
        "Name: {{ .Name }}",
        "{{- end }}",
        "",
    ];

    const filler = (i: number): string[] => [
        `{{ define "benchmark.filler_${i}" }}`,
        "{{- /*gotype: cg/model.Instance*/ -}}",
        "{{- $x := .Name -}}",
        "{{- $y := .Value -}}",
        "  Name: {{ $x }}",
        "  Value: {{ $y }}",
        `  Index: ${i}`,
        `  Info: static content for filler block ${i}`,
        "{{- end }}",
        "",
    ];

    const lines: string[] = [...header];
    let blockIdx = 0;
    while (lines.length < targetLines) {
        lines.push(...filler(blockIdx++));
    }
    return lines.slice(0, targetLines).join("\n") + "\n";
}

// ---- Statistics ----------------------------------------------------------

interface Stats {
    mean: number;
    p50: number;
    p95: number;
    min: number;
    max: number;
}

function computeStats(samples: number[]): Stats {
    const sorted = [...samples].sort((a, b) => a - b);
    const mean = samples.reduce((s, v) => s + v, 0) / samples.length;
    const pct = (p: number) => {
        const idx = Math.ceil((p / 100) * sorted.length) - 1;
        return sorted[Math.max(0, Math.min(idx, sorted.length - 1))] ?? 0;
    };
    return {
        mean,
        p50: pct(50),
        p95: pct(95),
        min: sorted[0] ?? 0,
        max: sorted[sorted.length - 1] ?? 0,
    };
}

function formatStats(s: Stats): string {
    return (
        `mean=${s.mean.toFixed(0)} ms  p50=${s.p50.toFixed(0)} ms  ` +
        `p95=${s.p95.toFixed(0)} ms  min=${s.min.toFixed(0)} ms  max=${s.max.toFixed(0)} ms`
    );
}

// ---- LSP helpers ---------------------------------------------------------

async function measureCompletion(
    uri: vscode.Uri,
    pos: vscode.Position,
): Promise<number> {
    const t0 = performance.now();
    await vscode.commands.executeCommand<vscode.CompletionList>(
        "vscode.executeCompletionItemProvider",
        uri,
        pos,
    );
    return performance.now() - t0;
}

async function measureDefinition(
    uri: vscode.Uri,
    pos: vscode.Position,
): Promise<number> {
    const t0 = performance.now();
    await vscode.commands.executeCommand<vscode.Location[]>(
        "vscode.executeDefinitionProvider",
        uri,
        pos,
    );
    return performance.now() - t0;
}

async function measureReferences(
    uri: vscode.Uri,
    pos: vscode.Position,
): Promise<number> {
    const t0 = performance.now();
    await vscode.commands.executeCommand<vscode.Location[]>(
        "vscode.executeReferenceProvider",
        uri,
        pos,
    );
    return performance.now() - t0;
}

/**
 * Polls the completion provider until it responds within `probeTimeoutMs`,
 * indicating the server has finished indexing the document.
 */
async function waitForServerReady(
    uri: vscode.Uri,
    pos: vscode.Position,
    probeTimeoutMs = 30_000,
    intervalMs = 500,
): Promise<void> {
    const deadline = Date.now() + probeTimeoutMs;
    while (Date.now() < deadline) {
        const t0 = performance.now();
        await vscode.commands.executeCommand<vscode.CompletionList>(
            "vscode.executeCompletionItemProvider",
            uri,
            pos,
        );
        const elapsed = performance.now() - t0;
        // Once the server responds fast enough (< 5 s) it is considered ready.
        if (elapsed < 5_000) return;
        await new Promise((r) => setTimeout(r, intervalMs));
    }
}

// ---- Suite ---------------------------------------------------------------

suite("Benchmark Suite", () => {
    let tmplUri: vscode.Uri;

    before(async function () {
        this.timeout(90_000); // allow generous time for fixture creation and server warmup

        const fixture = generateFixture(5_000);
        const result = await createDocument("benchmark.tmpl", fixture);
        tmplUri = result.tmplUri;

        // Wait for the language server to fully process the 5 000-line document.
        await waitForServerReady(tmplUri, COMPL_POS);
    });

    after(async () => {
        if (tmplUri) {
            await cleanupDocument(tmplUri);
        }
        vscode.window.showInformationMessage("Benchmark suite done.");
    });

    // ======================================================================
    // 1. Dynamic analysis - completions
    //    SLA: average ≤ 1 000 ms, p95 ≤ 2 000 ms
    // ======================================================================
    test("Dynamic analysis: average completion ≤ 1 000 ms and p95 ≤ 2 000 ms", async function () {
        this.timeout(120_000);

        for (let i = 0; i < WARMUP_ITERATIONS; i++) {
            await measureCompletion(tmplUri, COMPL_POS);
        }

        const times: number[] = [];
        for (let i = 0; i < BENCH_ITERATIONS; i++) {
            times.push(await measureCompletion(tmplUri, COMPL_POS));
        }

        const s = computeStats(times);
        console.log(`    Completion: ${formatStats(s)}`);

        assert.ok(
            s.mean <= THRESHOLD_AVERAGE_MS,
            `Completion average ${s.mean.toFixed(0)} ms exceeds threshold of ${THRESHOLD_AVERAGE_MS} ms`,
        );
        assert.ok(
            s.p95 <= THRESHOLD_P95_MS,
            `Completion p95 ${s.p95.toFixed(0)} ms exceeds threshold of ${THRESHOLD_P95_MS} ms`,
        );
    });

    // ======================================================================
    // 2. Jump to definition
    //    SLA: mean ≤ 1 000 ms
    // ======================================================================
    test(`Jump to definition: mean ≤ ${THRESHOLD_DEFINITION_MS} ms`, async function () {
        this.timeout(60_000);

        for (let i = 0; i < WARMUP_ITERATIONS; i++) {
            await measureDefinition(tmplUri, DEF_POS);
        }

        const times: number[] = [];
        for (let i = 0; i < BENCH_ITERATIONS; i++) {
            times.push(await measureDefinition(tmplUri, DEF_POS));
        }

        const s = computeStats(times);
        console.log(`    Definition: ${formatStats(s)}`);

        assert.ok(
            s.mean <= THRESHOLD_DEFINITION_MS,
            `Definition mean ${s.mean.toFixed(0)} ms exceeds threshold of ${THRESHOLD_DEFINITION_MS} ms`,
        );
    });

    // ======================================================================
    // 3. Find usages (references)
    //    SLA: mean ≤ 1 000 ms
    // ======================================================================
    test(`Find usages: mean ≤ ${THRESHOLD_REFERENCES_MS} ms`, async function () {
        this.timeout(60_000);

        for (let i = 0; i < WARMUP_ITERATIONS; i++) {
            await measureReferences(tmplUri, REF_POS);
        }

        const times: number[] = [];
        for (let i = 0; i < BENCH_ITERATIONS; i++) {
            times.push(await measureReferences(tmplUri, REF_POS));
        }

        const s = computeStats(times);
        console.log(`    References: ${formatStats(s)}`);

        assert.ok(
            s.mean <= THRESHOLD_REFERENCES_MS,
            `References mean ${s.mean.toFixed(0)} ms exceeds threshold of ${THRESHOLD_REFERENCES_MS} ms`,
        );
    });
});
