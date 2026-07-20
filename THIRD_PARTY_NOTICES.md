# Third-Party Notices

This project — the `text/template Support` VS Code extension and the
`Go-Text-Template` JetBrains plugin — is distributed under the MIT License
(see the `LICENSE` file in the repository root and the per-artifact license
files inside the packaged extension/plugin).

The distributed artifacts additionally include, in bundled or compiled form,
several third-party software components. The notices, copyright statements,
and license texts required by those components are reproduced below.

Sections:

1. [File-type icon (VS Code extension)](#1-file-type-icon-vs-code-extension)
2. [Forked source: Go standard library `text/template/parse`](#2-forked-source-go-standard-library-texttemplateparse)
3. [Go modules compiled into the language server binary](#3-go-modules-compiled-into-the-language-server-binary)
4. [npm packages bundled into the VS Code extension](#4-npm-packages-bundled-into-the-vs-code-extension)

The information below is the state at release time. To regenerate this file
against the current source tree, see the commands documented at the end
under [Regeneration](#regeneration).

---

## 1. File-type icon (VS Code extension)

The extension's file-type icon (`clients/VSCode/icons/icon.svg`) is derived
from `file_type_json.svg` in the
[vscode-icons](https://github.com/vscode-icons/vscode-icons) project.

Source: <https://github.com/vscode-icons/vscode-icons/blob/master/icons/file_type_json.svg>

```
The MIT License (MIT)

Copyright (c) 2016 Roberto Huertas

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

---

## 2. Forked source: Go standard library `text/template/parse`

The Go source files under `parse/` (notably `parse.go`, `lex.go`, `node.go`
and their helpers) are a fork of the `text/template/parse` package from the
Go standard library. Each of those files retains its original
`// Copyright 2011 The Go Authors. All rights reserved.` header. See
`parse/README.md` for a description of the modifications.

The Go project's `LICENSE` (BSD-3-Clause) — reproduced verbatim from
<https://go.googlesource.com/go/+/refs/heads/master/LICENSE> — applies:

```
Copyright (c) 2009 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google LLC nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

The corresponding `PATENTS` file from the Go project also applies:

```
Additional IP Rights Grant (Patents)

"This implementation" means the copyrightable works distributed by
Google as part of the Go project.

Google hereby grants to You a perpetual, worldwide, non-exclusive,
no-charge, royalty-free, irrevocable (except as stated in this section)
patent license to make, have made, use, offer to sell, sell, import,
transfer and otherwise run, modify and propagate the contents of this
implementation of Go, where such license applies only to those patent
claims, both currently owned or controlled by Google and acquired in
the future, licensable by Google that are necessarily infringed by this
implementation of Go.  This grant does not include claims that would be
infringed only as a consequence of further modification of this
implementation.  If you or your agent or exclusive licensee institute or
order or agree to the institution of patent litigation against any
entity (including a cross-claim or counterclaim in a lawsuit) alleging
that this implementation of Go or any code incorporated within this
implementation of Go constitutes direct or contributory patent
infringement, or inducement of patent infringement, then any patent
rights granted to you under this License for this implementation of Go
shall terminate as of the date such litigation is filed.
```

---

## 3. Go modules compiled into the language server binary

The `gotmpl-server*` binaries shipped inside the VS Code extension and the
JetBrains plugin are statically linked. The Go modules listed below are
imported by the server's production build (from `server/go.mod` and, via
the local `text-template-parser` replace, `parse/go.mod`) and their code is
therefore present in the distributed binaries. Each row identifies the
SPDX license identifier and the upstream repository, where the full
`LICENSE` (and, where present, `NOTICE` / `PATENTS`) text can be obtained.

For the `golang.org/x/*` modules, the additional patent grant is the same
Go project `PATENTS` text already reproduced in section 2.

Test-only dependencies (compiled only into `*_test.go` binaries and not
into the shipped `gotmpl-server*` executables) are omitted.

| Module | License | Upstream |
| --- | --- | --- |
| github.com/rs/zerolog | MIT | https://github.com/rs/zerolog |
| github.com/tliron/glsp | Apache-2.0 | https://github.com/tliron/glsp |
| github.com/tliron/commonlog | Apache-2.0 | https://github.com/tliron/commonlog |
| github.com/tliron/kutil | Apache-2.0 | https://github.com/tliron/kutil |
| github.com/sourcegraph/jsonrpc2 | MIT | https://github.com/sourcegraph/jsonrpc2 |
| github.com/gorilla/websocket | BSD-2-Clause | https://github.com/gorilla/websocket |
| github.com/segmentio/ksuid | MIT | https://github.com/segmentio/ksuid |
| github.com/iancoleman/strcase | MIT | https://github.com/iancoleman/strcase |
| github.com/pkg/errors | BSD-2-Clause | https://github.com/pkg/errors |
| github.com/petermattis/goid | Apache-2.0 | https://github.com/petermattis/goid |
| github.com/sasha-s/go-deadlock | Apache-2.0 | https://github.com/sasha-s/go-deadlock |
| github.com/muesli/termenv | MIT | https://github.com/muesli/termenv |
| github.com/aymanbagabas/go-osc52/v2 | MIT | https://github.com/aymanbagabas/go-osc52 |
| github.com/lucasb-eyer/go-colorful | MIT | https://github.com/lucasb-eyer/go-colorful |
| github.com/mattn/go-colorable | MIT | https://github.com/mattn/go-colorable |
| github.com/mattn/go-isatty | MIT | https://github.com/mattn/go-isatty |
| github.com/rivo/uniseg | MIT | https://github.com/rivo/uniseg |
| go.lsp.dev/uri | BSD-3-Clause | https://github.com/go-language-server/uri |
| golang.org/x/tools | BSD-3-Clause | https://cs.opensource.google/go/x/tools |
| golang.org/x/crypto | BSD-3-Clause | https://cs.opensource.google/go/x/crypto |
| golang.org/x/mod | BSD-3-Clause | https://cs.opensource.google/go/x/mod |
| golang.org/x/net | BSD-3-Clause | https://cs.opensource.google/go/x/net |
| golang.org/x/sync | BSD-3-Clause | https://cs.opensource.google/go/x/sync |
| golang.org/x/sys | BSD-3-Clause | https://cs.opensource.google/go/x/sys |
| golang.org/x/term | BSD-3-Clause | https://cs.opensource.google/go/x/term |
| gopkg.in/yaml.v3 | MIT and Apache-2.0 | https://github.com/go-yaml/yaml |

---

## 4. npm packages bundled into the VS Code extension

`clients/VSCode/esbuild.js` builds `out/extension.js` with `bundle: true`,
inlining the transitive closure of runtime imports starting from
`src/extension.ts`. The only runtime import outside `vscode` itself is
`vscode-languageclient/node`, which pulls in the packages below. Their code
is therefore present in the shipped `.vsix`. All are permissive
(MIT / ISC).

| Package | License | Upstream |
| --- | --- | --- |
| vscode-languageclient | MIT | https://github.com/microsoft/vscode-languageserver-node |
| vscode-languageserver-protocol | MIT | https://github.com/microsoft/vscode-languageserver-node |
| vscode-languageserver-types | MIT | https://github.com/microsoft/vscode-languageserver-node |
| vscode-jsonrpc | MIT | https://github.com/microsoft/vscode-languageserver-node |
| semver | ISC | https://github.com/npm/node-semver |
| minimatch | ISC | https://github.com/isaacs/minimatch |
| brace-expansion | MIT | https://github.com/juliangruber/brace-expansion |
| balanced-match | MIT | https://github.com/juliangruber/balanced-match |

The following packages appear as `dependencies` in
`clients/VSCode/package.json` but are only used from `src/test/**` and are
therefore **not** inlined into the shipped `out/extension.js`:
`vscode-oniguruma` (MIT), `vscode-textmate` (MIT). They are noted here for
completeness.

All other npm packages (`devDependencies` such as `@vscode/vsce`,
`eslint`, `typescript`, `esbuild`, `@vscode/test-*`) are used only at
build / lint / test time on developer machines and are not present in any
distributed artifact.

---

## Verification

To regenerate or verify this listing against the current source tree,
the following upstream tools are conventional:

- `go-licenses report ./...` from `server/` — inventory of Go module
  licenses actually compiled into the server binary.
- `npx license-checker-rseidelsohn --production --direct 0` from
  `clients/VSCode/` — inventory of npm licenses in the runtime dependency
  closure.

