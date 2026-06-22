# zftp v2 Modernization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Relicense, modernize, and unit-test the zftp z/OS FTP client to v2 so a bank/insurer can adopt it: Apache-2.0, concrete-typed API, table-driven parsers proven against fixtures, an in-process mock z/OS server enabling real unit tests, and gated pure-Go CI.

**Architecture:** Keep the existing package layout (`zftp` root, `hfs`, `eol`, `internal/*`). Add a connection seam (`WithDialer` + unexported `newSession`) and `internal/mockzos` (a real loopback FTP server) so the actual client code is tested end-to-end without a mainframe. Rewrite the `hfs` parsers as declarative field tables with row classifiers. Reserve a separate-module `cli/` slot to keep the library zero-dep.

**Tech Stack:** Go 1.20, stdlib only (net, crypto/tls, bufio), `staticcheck`, `govulncheck`, GitHub Actions.

## Global Constraints

- Module path: `gopkg.in/ro-ag/zftp.v2` (verbatim).
- Pure-Go: `CGO_ENABLED=0 go build ./...` and `CGO_ENABLED=0 go vet ./...` must pass. No cgo in source.
- `go test -race ./...` must pass (race lane uses `CGO_ENABLED=1`; that is tooling, not source cgo).
- License: Apache-2.0; every `.go` file begins with `// SPDX-License-Identifier: Apache-2.0`.
- Library `go.mod` stays free of external `require`s (zero-dep). CLI deps live only in `cli/go.mod`.
- Do not regress protocol behavior: ASCII/binary type switch + restore, EOL conversion (CRLF on ASCII store), passive mode, `REST` offset resume, multiline reply parsing, `ReturnError` semantics.
- Fixtures honesty: real captures are ground truth; spec-derived fixtures are header-labeled as such; never present synthesized output as a real capture.
- Frequent commits (one per task minimum). Conventional Commit messages.

## File Structure

**Created:**
- `LICENSE` (rewritten), `NOTICE`
- `internal/mockzos/server.go` — loopback FTP server (control + PASV data + scripts)
- `internal/mockzos/script.go` — response/dataset scripting helpers
- `hfs/layout.go` — generic field-table engine (`field`, `slice`, classifiers)
- `hfs/dataset_test.go`, `hfs/partitioned_test.go`, `hfs/spool_test.go` — golden tests (extend existing)
- `dialer.go` — `Dialer` type + `WithDialer` option + `newSession` seam
- `cmd_test.go`, `codes_test.go`, `transfer_test.go`, `passive_test.go`, `lists_test.go`, `xstat_test.go` (in-process, replacing/augmenting env-gated tests)
- `.github/workflows/ci.yml`, `.github/dependabot.yml`
- `cli/go.mod`, `cli/main.go`, `cli/README.md` (skeleton only)
- `docs/superpowers/plans/2026-06-21-zftp-v2-modernization.md` (this file)

**Modified:**
- `go.mod` (module path), every `.go` with a `.v1` import (→ `.v2`) and SPDX header
- `xstat.go`, `site.go` — `StatusOf()`/`SetStatusOf()` return concrete types
- `transfer.go` — `TransferType` → concrete `Type` enum
- `hfs/dataset.go`, `hfs/partitioned.go`, `hfs/spool.go` — table-driven rewrites
- `Readme.md` — `.v0`→`.v2`, runnable example, accurate claims, badges

---

## Task 0: Baseline snapshot (safety net before refactor)

**Files:** none created; captures current behavior.

- [ ] **Step 1:** Confirm clean build on the branch.

Run: `CGO_ENABLED=0 go build ./... && go test ./hfs/... 2>&1 | tail -20`
Expected: build succeeds; existing `hfs` parser tests pass (these are the current golden behavior we must preserve).

- [ ] **Step 2:** Record the current public API surface for diffing later.

Run: `go doc -all . > /tmp/zftp_api_before.txt; go doc -all ./hfs >> /tmp/zftp_api_before.txt; wc -l /tmp/zftp_api_before.txt`
Expected: a file capturing the v1 API (reference only; not committed).

- [ ] **Step 3:** No commit (read-only baseline).

---

## Task ZM01: Apache-2.0 relicense + NOTICE + SPDX headers

**Files:**
- Modify: `LICENSE`
- Create: `NOTICE`
- Modify: every `*.go` (prepend SPDX header)
- Modify: `Readme.md` (license section/badge)

**Interfaces:** none (textual/legal).

- [ ] **Step 1:** Replace `LICENSE` with the verbatim Apache License 2.0 text from `https://www.apache.org/licenses/LICENSE-2.0.txt`. The appendix's `[yyyy]`/`[name]` boilerplate stays as the standard instructional text (Apache convention) — actual attribution goes in `NOTICE`.

- [ ] **Step 2:** Create `NOTICE`:

```
zftp
Copyright 2023 Rodrigo Agurto

This product includes software developed by Rodrigo Agurto.
Licensed under the Apache License, Version 2.0.
```

- [ ] **Step 3:** Add SPDX headers to all current `.go` files (build-tag safe: SPDX line, blank line, then existing content, so any `//go:build` constraint remains valid):

```bash
for f in $(git ls-files '*.go'); do
  head -1 "$f" | grep -q 'SPDX-License-Identifier' && continue
  printf '// SPDX-License-Identifier: Apache-2.0\n\n%s' "$(cat "$f")" > "$f.tmp" && mv "$f.tmp" "$f"
done
gofmt -w .
```

- [ ] **Step 4:** Verify build + that headers didn't break build constraints.

Run: `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 GOOS=windows go build ./eol/ && CGO_ENABLED=0 GOOS=linux go build ./eol/`
Expected: all succeed (confirms `eol_windows.go`/`eol_unix.go` build tags survived).

- [ ] **Step 5:** Update `Readme.md` license section to Apache-2.0 + add badge `![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)`.

- [ ] **Step 6:** Commit.

```bash
git add LICENSE NOTICE Readme.md '*.go'
git commit -m "license: relicense to Apache-2.0 with NOTICE and SPDX headers"
```

---

## Task ZM02: v2 module bump + import rewrite

**Files:** Modify `go.mod` + every `.go` importing `zftp.v1`.

**Interfaces:** Produces the `.v2` import path all later tasks use.

- [ ] **Step 1:** Rewrite the module path and all internal imports (macOS `sed -i ''`):

```bash
sed -i '' 's#^module gopkg.in/ro-ag/zftp.v1#module gopkg.in/ro-ag/zftp.v2#' go.mod
for f in $(grep -rl 'gopkg.in/ro-ag/zftp.v1' --include='*.go' .); do
  sed -i '' 's#gopkg.in/ro-ag/zftp\.v1#gopkg.in/ro-ag/zftp.v2#g' "$f"
done
```

- [ ] **Step 2:** Verify no stragglers.

Run: `grep -rn 'zftp\.v1' --include='*.go' . ; grep -n 'zftp.v1' go.mod`
Expected: no output.

- [ ] **Step 3:** Build + existing tests.

Run: `CGO_ENABLED=0 go build ./... && go test ./hfs/...`
Expected: PASS.

- [ ] **Step 4:** Commit.

```bash
git add go.mod '*.go'
git commit -m "build!: bump module path to gopkg.in/ro-ag/zftp.v2"
```

---

## Task ZM03: Parser rewrite — table-driven field engine + golden fixtures

> Done before session changes because it is self-contained (`hfs` package, pure functions) and establishes the declarative pattern. **Test-first against the existing real captures** so the refactor cannot regress.

**Files:**
- Create: `hfs/layout.go`
- Modify: `hfs/dataset.go`, `hfs/partitioned.go`, `hfs/spool.go`
- Modify/extend: `hfs/dataset_test.go`, `hfs/partitioned_test.go`, `hfs/spool_test.go`
- Fixtures: existing `hfs/*_test.txt` (ground truth) + new labeled cases

**Interfaces:**
- Produces: `ParseInfoDataset(string) (InfoDataset, error)`, `ParseInfoPdsMember(string) (InfoPdsMember, error)`, `ParseInfoJob([]string) ([]InfoJob, error)`, `ParseInfoJobDetail([]string) (*InfoJobDetail, error)` — **same signatures and output shapes as v1** (no consumer-visible change; internals become table-driven).

- [ ] **Step 1: Pin current behavior (characterization test).** Before changing parser internals, add a golden test that parses each existing real fixture and asserts the full struct via a stable serialization, capturing today's correct output.

```go
// hfs/dataset_test.go
func TestParseInfoDataset_GoldenFixture(t *testing.T) {
    lines := readFixtureLines(t, "dataset_test.txt") // existing real capture
    for i, line := range lines[1:] { // skip header row
        got, err := ParseInfoDataset(line)
        if err != nil { t.Fatalf("line %d: %v", i, err) }
        snap := toJSON(t, got)
        golden.Assert(t, snap, fmt.Sprintf("dataset_line_%02d", i))
    }
}
```

- [ ] **Step 2:** Run to generate/verify goldens against current parser.

Run: `go test ./hfs/ -run GoldenFixture -update && go test ./hfs/ -run GoldenFixture`
Expected: PASS (goldens now encode v1's correct output — the contract for the refactor).

- [ ] **Step 3:** Add the field-table engine `hfs/layout.go`:

```go
package hfs

// field describes one fixed-width column. width 0 means "to end of line".
type field struct {
    name  string
    start int
    width int
}

// slice returns the trimmed substring for f, tolerant of short lines.
func (f field) slice(rec string) string {
    if f.start >= len(rec) { return "" }
    end := len(rec)
    if f.width > 0 && f.start+f.width < end { end = f.start + f.width }
    return rec[f.start:end]
}

// classifier inspects a raw record and returns a non-empty tag for special
// rows (e.g. "migrated", "notmounted") or "" for a normal record.
type classifier func(rec string) string
```

- [ ] **Step 4:** Rewrite `ParseInfoDataset` over a declarative layout + classifiers, preserving exact field semantics (volume/unit/referred/ext/used/recfm/lrecl/blksize/dsorg/dsname, plus `Migrated`/`Not Mounted` and the BLKSIZE-overflow case from commit `64753fc`). Keep `FieldString`/`FieldInt`/etc. outputs identical.

```go
var datasetLayout = []field{
    {"Volume", 0, 6}, {"Unit", 6, 5}, {"Referred", 11, 13}, {"Ext", 24, 3},
    {"Used", 27, 5}, {"Recfm", 32, 6}, {"Lrecl", 38, 6}, {"BlkSz", 44, 6},
    {"Dsorg", 51, 5}, {"Dsname", 56, 0},
}
// classifiers run before layout slicing; see dataset.go for migrated/notmounted/overflow handling.
```

- [ ] **Step 5:** Run goldens — refactor must match the pinned output exactly.

Run: `go test ./hfs/ -run GoldenFixture`
Expected: PASS (no diff vs Step 2 goldens).

- [ ] **Step 6:** Repeat Steps 1–5 for `ParseInfoPdsMember` (`hfs/partitioned.go`) and the JES parsers (`hfs/spool.go`), each pinned then rewritten table-driven. Preserve interface-level-1/2 detection, `ACTIVE`/ABEND/JCL-error, RC extraction.

- [ ] **Step 7: Add labeled spec-derived + negative fixtures.** For documented cases without a real capture (e.g., GDG, VSAM, multi-volume, very large LRECL/BLKSIZE), add fixtures whose first line is a comment `# spec-derived (IBM LIST format), not a live capture` and a golden per case. Add malformed inputs asserting specific errors:

```go
func TestParseInfoDataset_Malformed(t *testing.T) {
    _, err := ParseInfoDataset("too short")
    if err == nil { t.Fatal("want error for short record") }
}
```

- [ ] **Step 8:** Full hfs suite.

Run: `go test -race ./hfs/...`
Expected: PASS.

- [ ] **Step 9:** Commit.

```bash
git add hfs/
git commit -m "feat(hfs)!: table-driven parsers with golden fixtures and negative tests"
```

---

## Task ZM04: Connection seam (`WithDialer` + `newSession`)

**Files:**
- Create: `dialer.go`
- Modify: `ftp.go` (`Open` uses the seam), `options.go` (register option)

**Interfaces:**
- Produces: `type Dialer interface { DialContext(ctx context.Context, network, addr string) (net.Conn, error) }`; `func WithDialer(d Dialer) Option`; unexported `func newSession(conn net.Conn) *FTPSession`.
- Consumes: existing `Open(server string, opts ...Option) (*FTPSession, error)`.

- [ ] **Step 1: Failing test** — inject a dialer that hands back one end of `net.Pipe`, drive a canned greeting, assert `Open` reads it.

```go
// dialer_test.go
func TestOpen_WithInjectedDialer(t *testing.T) {
    cli, srv := net.Pipe()
    go func() { io.WriteString(srv, "220 Service ready\r\n") ; /* ... */ }()
    s, err := Open("fake:21", WithDialer(pipeDialer{cli}))
    if err != nil { t.Fatal(err) }
    _ = s.Close()
}
```

- [ ] **Step 2:** Run → FAIL (`WithDialer` undefined).

Run: `go test ./ -run WithInjectedDialer`
Expected: compile error / FAIL.

- [ ] **Step 3:** Implement `dialer.go`: define `Dialer`, default to `&net.Dialer{Timeout, KeepAlive}` from `dialCfg`, add `WithDialer`. Refactor `Open` to obtain the conn via the configured dialer, then call `newSession(conn)` (extracted from current `Open` body).

- [ ] **Step 4:** Run → PASS.

Run: `go test -race ./ -run WithInjectedDialer`
Expected: PASS.

- [ ] **Step 5:** Commit.

```bash
git add dialer.go ftp.go options.go dialer_test.go
git commit -m "feat: add WithDialer connection seam for testability"
```

---

## Task ZM05: `internal/mockzos` — in-process z/OS FTP server

**Files:**
- Create: `internal/mockzos/server.go`, `internal/mockzos/script.go`, `internal/mockzos/server_test.go`

**Interfaces:**
- Produces:
  - `func New(t testing.TB, opts ...Option) *Server`
  - `func (*Server) Addr() string`
  - `func (*Server) Close() error`
  - `func (*Server) Script(cmd string, replies ...string)` — canned control replies
  - `func (*Server) DataFor(cmd, expr string, payload string)` — payload streamed over PASV data conn for a matching `LIST`/`RETR`/etc.
- Consumes: nothing from the library (avoids import cycle; lives in `internal/`).

- [ ] **Step 1: Failing test** — server answers a control command.

```go
func TestServer_GreetingAndUser(t *testing.T) {
    s := New(t)
    c, _ := net.Dial("tcp", s.Addr())
    defer c.Close()
    if got := readLine(t, c); !strings.HasPrefix(got, "220") { t.Fatalf("greeting: %q", got) }
}
```

- [ ] **Step 2:** Run → FAIL.

Run: `go test ./internal/mockzos/ -run GreetingAndUser`
Expected: FAIL.

- [ ] **Step 3:** Implement the server: TCP listener on `127.0.0.1:0`; per-connection goroutine; default handlers for `USER`/`PASS`→`230`, `SYST`→`215 MVS ...`, `TYPE`→`200`, `FEAT`/`QUIT`; multiline reply writer; `PASV` handler that opens a second listener, advertises `227 ...(h1,h2,h3,h4,p1,p2)`, accepts the data conn, writes the scripted payload, closes it, and sends `226`. Register cleanup via `t.Cleanup`.

- [ ] **Step 4:** Run → PASS; add a PASV data-transfer test (script a `LIST` payload, dial, read `150`/data/`226`).

Run: `go test -race ./internal/mockzos/...`
Expected: PASS.

- [ ] **Step 5:** Commit.

```bash
git add internal/mockzos/
git commit -m "test: add in-process mock z/OS FTP server (control + PASV data)"
```

---

## Task ZM06: API cleanup — concrete returns + concrete `Type` enum

**Files:**
- Modify: `xstat.go` (`StatusOf`), `site.go` (`SetStatusOf`), `transfer.go` (`TransferType`), `internal/helper/*` (backing impls), all call sites + tests.

**Interfaces:**
- Produces:
  - `func (s *FTPSession) StatusOf() *ServerStatus` (was `StatusOf` interface)
  - `func (s *FTPSession) SetStatusOf() *StatusSetter` (was `StatusUpdater` interface)
  - `type Type` concrete enum with `TypeAscii`, `TypeBinary` and methods `String()`, `IsAscii()`, `IsBinary()`, internal `command()`; `SetType(Type)`, `Get(..., Type)`, etc. updated.

- [ ] **Step 1: Pin behavior** — with `mockzos` (ZM05), add a test asserting one `StatusOf` getter and one `SetStatusOf` setter produce the correct `XSTA`/`SITE` command and parse the canned reply, using the **current** interface return. This locks behavior pre-refactor.

```go
func TestServerStatus_Blocksize(t *testing.T) {
    s, srv := dialMock(t)            // helper: Open+Login against mockzos
    srv.Script("XSTA BLOCKSIZE", "211 BLOCKSIZE 27998")
    got, err := s.StatusOf().BlockSize()
    if err != nil { t.Fatal(err) }
    if got != 27998 { t.Fatalf("got %d", got) }
}
```

- [ ] **Step 2:** Run → PASS against current interface.

- [ ] **Step 3:** Convert returns to concrete: export the backing struct as `ServerStatus`/`StatusSetter` (move from `internal/helper` to root or alias), change accessor signatures, **delete** the `StatusOf`/`StatusUpdater` interface declarations. Replace `TransferType` interface with concrete `Type` (carry `IsAscii`/`IsBinary`/`String`/`command`); update `transfer.go`, `get.go`, `put.go`, `lists.go` signatures and the `TypeAscii`/`TypeBinary` exported values.

- [ ] **Step 4:** Run the pinned tests + full build → PASS, and diff the API surface.

Run: `go test -race ./ -run 'ServerStatus|Type' && go doc -all . | grep -E 'func .*StatusOf|TransferType|type Type'`
Expected: tests PASS; `go doc` shows concrete `*ServerStatus`/`*StatusSetter` returns and concrete `Type`; no `StatusOf`/`TransferType` interface remains.

- [ ] **Step 5:** Commit.

```bash
git add -A
git commit -m "refactor!: return concrete types (*ServerStatus, *StatusSetter, Type) instead of interfaces"
```

---

## Task ZM07: Unit test suite against mockzos (`-race`)

**Files:**
- Create/replace: `cmd_test.go`, `codes_test.go`, `transfer_test.go`, `passive_test.go`, `lists_test.go`, `xstat_test.go`; add `testhelpers_test.go` (`dialMock`).
- Keep env-gated integration tests skipping when `ZFTP_*` unset.

**Interfaces:** Consumes `mockzos` (ZM05), seam (ZM04), concrete API (ZM06).

- [ ] **Step 1:** `testhelpers_test.go`: `dialMock(t)` → starts `mockzos`, `Open`s with default dialer to `srv.Addr()`, `Login`s, returns `(*FTPSession, *mockzos.Server)`.

- [ ] **Step 2: codes/errors** — multiline reply parsing + `ReturnError`:

```go
func TestSendCommand_MultilineAndReturnError(t *testing.T) {
    s, srv := dialMock(t)
    srv.Script("STAT", "211-begin", "211 end")
    if _, err := s.SendCommand(CodeFileStatus, "STAT"); err != nil { t.Fatal(err) }
    srv.Script("STOR x", "550 No.")
    _, err := s.SendCommand(CodeFileActionOK, "STOR x")
    var re *ReturnError
    if !errors.As(err, &re) || re.ReturnCode() != 550 { t.Fatalf("want 550 ReturnError, got %v", err) }
}
```

- [ ] **Step 3: transfer/EOL** — ASCII store appends CRLF; binary is byte-exact; type restored after transfer; `RetrieveIO` to a buffer matches scripted payload. (Use `mockzos.DataFor`.)

- [ ] **Step 4: passive** — valid `227` parsed to correct port; malformed `227` returns an error.

- [ ] **Step 5: datasets/lists** — `ListDatasets`/`ListPds`/`ListSpool` over scripted listing payloads return parsed structs equal to the `hfs` golden expectations; `SetDataSpecs` emits the exact `SITE RECFM=.. LRECL=.. BLKSIZE=..` string.

- [ ] **Step 6:** Run the whole suite with race.

Run: `go test -race ./...`
Expected: PASS.

- [ ] **Step 7:** Report coverage.

Run: `go test -cover ./... | sort`
Expected: meaningful coverage on `cmd.go`, `codes.go`, `transfer.go`, `passive.go`, `lists.go`, `hfs/*`.

- [ ] **Step 8:** Commit.

```bash
git add '*_test.go'
git commit -m "test: in-process unit suite for commands, codes, transfer/EOL, passive, datasets (-race)"
```

---

## Task ZM08: CI + Dependabot

**Files:** Create `.github/workflows/ci.yml`, `.github/dependabot.yml`.

- [ ] **Step 1:** Write `ci.yml` (SHA-pin `actions/checkout`, `actions/setup-go`; resolve real SHAs at authoring time and annotate with the version):

```yaml
name: CI
on:
  push: { branches: [main, v2-modernization] }
  pull_request:
permissions: { contents: read }
concurrency: { group: "${{ github.workflow }}-${{ github.ref }}", cancel-in-progress: true }
jobs:
  build-vet:
    strategy: { matrix: { os: [ubuntu-latest, macos-latest] } }
    runs-on: ${{ matrix.os }}
    env: { CGO_ENABLED: 0 }
    steps:
      - uses: actions/checkout@<sha> # v4.x
      - uses: actions/setup-go@<sha> # v5.x
        with: { go-version: '1.20' }
      - run: go build ./...
      - run: go vet ./...
      - run: go install honnef.co/go/tools/cmd/staticcheck@latest && staticcheck ./...
  race:
    strategy: { matrix: { os: [ubuntu-latest, macos-latest] } }
    runs-on: ${{ matrix.os }}
    env: { CGO_ENABLED: 1 }
    steps:
      - uses: actions/checkout@<sha> # v4.x
      - uses: actions/setup-go@<sha> # v5.x
        with: { go-version: '1.20' }
      - run: go test -race ./...
  vuln:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@<sha> # v4.x
      - uses: actions/setup-go@<sha> # v5.x
        with: { go-version: '1.20' }
      - run: go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...
  ci-success:
    needs: [build-vet, race, vuln]
    runs-on: ubuntu-latest
    steps: [{ run: 'echo "all green"' }]
```

- [ ] **Step 2:** Write `dependabot.yml`:

```yaml
version: 2
updates:
  - package-ecosystem: gomod
    directory: "/"
    schedule: { interval: weekly }
    groups: { gomod: { patterns: ["*"] } }
  - package-ecosystem: gomod
    directory: "/cli"
    schedule: { interval: weekly }
    groups: { cli: { patterns: ["*"] } }
  - package-ecosystem: github-actions
    directory: "/"
    schedule: { interval: weekly }
    groups: { actions: { patterns: ["*"] } }
```

- [ ] **Step 3:** Lint locally before relying on CI.

Run: `CGO_ENABLED=0 go vet ./... && go run honnef.co/go/tools/cmd/staticcheck@latest ./... ; go run golang.org/x/vuln/cmd/govulncheck@latest ./...`
Expected: clean (fix findings; common: error-string capitalization, unused).

- [ ] **Step 4:** Commit + push; confirm CI green on the branch.

```bash
git add .github/
git commit -m "ci: pure-Go GitHub Actions matrix with race, staticcheck, govulncheck, gate, Dependabot"
git push
```

---

## Task ZM09-slot: CLI-ready structure (skeleton only)

**Files:** Create `cli/go.mod`, `cli/main.go`, `cli/README.md`.

- [ ] **Step 1:** `cli/go.mod` as a separate module with a local `replace` so it builds against the parent during dev:

```
module github.com/ro-ag/zftp/cli

go 1.20

require gopkg.in/ro-ag/zftp.v2 v2.0.0
replace gopkg.in/ro-ag/zftp.v2 => ../
```

- [ ] **Step 2:** Minimal `cli/main.go` proving isolation (imports the library, prints version) — not a functional CLI:

```go
// SPDX-License-Identifier: Apache-2.0
package main

import (
    "fmt"
    _ "gopkg.in/ro-ag/zftp.v2"
)

const version = "0.0.0-dev"

func main() { fmt.Println("zftp cli (skeleton)", version) }
```

- [ ] **Step 3:** Verify the library stays zero-dep and the CLI builds in isolation.

Run: `go mod graph | grep -v '^gopkg.in/ro-ag/zftp.v2 ' | grep -c . ; (cd cli && CGO_ENABLED=0 go build ./...)`
Expected: library graph shows no external deps; CLI builds.

- [ ] **Step 4:** `cli/README.md` documents the follow-up (`ZM09`: cobra + GoReleaser binaries). Commit.

```bash
git add cli/
git commit -m "chore(cli): reserve separate-module CLI slot, keep library zero-dep"
```

---

## Task ZM10: README + GoDoc fixes + runnable example

**Files:** Modify `Readme.md`; add `example_test.go` (compilable example).

- [ ] **Step 1:** Replace every `.v0`/`.v1` in badge, doc link, install, and example with `.v2`. GoDoc link → `https://pkg.go.dev/gopkg.in/ro-ag/zftp.v2`.

- [ ] **Step 2:** Add a compilable `Example` so the README snippet is test-verified:

```go
// example_test.go
func Example() {
    s, err := zftp.Open("mainframe.example.com:21")
    if err != nil { log.Fatal(err) }
    defer s.Close()
    if err := s.Login("USER", "PASS"); err != nil { log.Fatal(err) }
    ds, err := s.ListDatasets("USER.*")
    if err != nil { log.Fatal(err) }
    fmt.Println(len(ds))
}
```

- [ ] **Step 3:** Tighten claims: pure-Go, zero external deps, Apache-2.0, z/OS-specialized (datasets/JES/HFS/SITE/passive/ASCII-binary+EOL). Remove anything unverifiable.

- [ ] **Step 4:** Verify example compiles (it builds with the package).

Run: `go build ./... && go vet ./...`
Expected: PASS.

- [ ] **Step 5:** Commit.

```bash
git add Readme.md example_test.go
git commit -m "docs: fix GoDoc links to v2, add runnable example, accurate claims"
```

---

## Task ZM11: Final verification

- [ ] **Step 1:** Full gate locally; capture output for the PR.

```bash
CGO_ENABLED=0 go build ./... && \
CGO_ENABLED=0 go vet ./... && \
go run honnef.co/go/tools/cmd/staticcheck@latest ./... && \
go test -race ./... && \
go run golang.org/x/vuln/cmd/govulncheck@latest ./... && \
(cd cli && CGO_ENABLED=0 go build ./...)
```
Expected: all PASS.

- [ ] **Step 2:** Open the PR (body = summary + verification output + closes `ZM01–ZM10`). Confirm CI green.

- [ ] **Step 3:** API diff sanity: `go doc -all .` shows concrete returns, no removed-by-accident exports beyond the intended interface removals.

---

## Self-Review (plan vs spec)

- **Spec §0 versioning** → ZM02. **§1 license** → ZM01. **§2 API** → ZM06. **§3 seam+mock** → ZM04+ZM05. **§4 parsers** → ZM03. **§5 tests** → ZM07. **§6 CI** → ZM08. **§7 README** → ZM10. **§8 CLI-ready** → ZM09-slot. **§9 delivery** → issues+project (orchestration). **§10 verify** → ZM11. No gaps.
- **Ordering note:** ZM03 (parsers) precedes ZM06 (API) intentionally — parsers are independent; pinning their goldens first de-risks. ZM04/ZM05 precede ZM06/ZM07 because tests need the seam + mock.
- **Placeholders:** none — `<sha>` in CI is resolved at authoring time (documented), not a logic gap.
- **Type consistency:** `*ServerStatus`, `*StatusSetter`, `Type`/`TypeAscii`/`TypeBinary`, `Dialer`/`WithDialer`/`newSession`, `mockzos.New/Addr/Script/DataFor` used consistently across tasks.
