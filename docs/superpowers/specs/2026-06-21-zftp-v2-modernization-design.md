# zftp v2 Modernization — Design Spec

- **Date:** 2026-06-21
- **Status:** Draft (awaiting review)
- **Module:** `gopkg.in/ro-ag/zftp` — pure-Go FTP client specialized for z/OS
- **Current:** `gopkg.in/ro-ag/zftp.v1` @ v1.3.1, MIT, Go 1.20, zero external deps
- **Target:** `gopkg.in/ro-ag/zftp.v2`, Apache-2.0, idiomatic, unit-tested, CI-gated

## Goal

Make zftp idiomatic, well-tested, and trustworthy enough for enterprise (bank /
insurer) adoption where legal teams and procurement vet dependencies. Clean
licensing, real test coverage, and protocol correctness matter more than new
features. Stay pure-Go / no cgo in source. Do not regress protocol behavior
(ASCII/binary transfer, EOL handling, return-code semantics).

## Non-Goals (this effort)

- Building the full CLI binary + release pipeline (reserved as follow-up `ZM09`).
  This effort only reserves the structure so the library stays zero-dep.
- New protocol features beyond what v1 already supports.
- Supporting both v1 and v2 in parallel beyond the frozen v1 tags.

## Constraints

- Pure-Go, `CGO_ENABLED=0` must build and pass `go vet`.
- `go test -race ./...` must pass. (Note: the race detector itself requires cgo;
  see CI lanes below. This does not introduce cgo into library source.)
- Protocol correctness is non-negotiable; parser rewrites are proven against
  real-output fixtures with exact-match golden tests.

---

## 0. Versioning & module mechanics

- `go.mod`: `module gopkg.in/ro-ag/zftp.v2`, keep `go 1.20` (or bump only if a
  feature requires it; decide during implementation, default keep).
- Rewrite every internal import `gopkg.in/ro-ag/zftp.v1/...` → `.v2/...`.
- Existing `v1.*` tags remain frozen and continue to serve current v1 users via
  gopkg.in (tags are immutable; their `go.mod` still says `.v1`). No v1 branch
  maintenance planned (adoption ~0).
- `main` becomes the v2 line. Release = tag `v2.0.0`; gopkg.in then resolves
  `gopkg.in/ro-ag/zftp.v2`. Tagging is the maintainer's step, not part of the PR.
- **Hard break, no compatibility shims** (per decision: ~0 adoption).

**Acceptance:** repo builds under the new module path; `grep -r 'zftp.v1'` returns
only historical references (none in code/imports).

---

## 1. License — Apache-2.0 (do first)

Without a clear license the library is legally unusable for the target audience;
this lands first.

- Replace `LICENSE` contents: MIT → full Apache-2.0 text.
- Add `NOTICE`: `Copyright 2023 Rodrigo Agurto` (+ project line). Apache §4(d)
  attribution home; recognized by enterprise license scanners.
- Add SPDX header to every `.go` file:
  `// SPDX-License-Identifier: Apache-2.0`
- Update README license section + add license badge.

Sole authorship verified (`git shortlog -sne` → only Rodrigo Agurto identities),
so relicensing is clean with no third-party copyright to clear.

**Acceptance:** `LICENSE` is verbatim Apache-2.0; `NOTICE` present; every `.go`
file carries the SPDX header; README states Apache-2.0.

---

## 2. API cleanup — "accept interfaces, return structs"

Return concrete types from constructors/accessors. Keep interfaces only at narrow
input boundaries.

| Symbol | Current | Action |
|---|---|---|
| `StatusOf() StatusOf` (100+ getters) | returns interface | Return concrete `*ServerStatus`. Methods stay (each queries the server), type becomes concrete. |
| `SetStatusOf() StatusUpdater` (~11) | returns interface | Return concrete `*StatusSetter`. |
| `TransferType` (behavioral iface) | param interface, private impls | Convert to concrete enum type `Type` with exported values `TypeAscii`, `TypeBinary` (+ `String()`, `IsAscii()`, `IsBinary()`). |
| `DataSpec` (`Apply() (string,error)`) | input option iface | Keep — narrow input boundary (functional-option style). |
| `JesSpec` (`Apply(*FTPSession) error`) | input option iface | Keep — narrow input boundary. |
| `eol.LineBreaker` (`String`,`NewLine`) | input iface, 3 consts | Keep — narrow input, enum-like. |
| `Option` (func type) | functional option | Keep. |
| `internal/transfer.DataTransfer` | internal strategy iface | Keep (internal seam, not public API). |

Naming for the new concrete status types is provisional (`ServerStatus`,
`StatusSetter`); finalize during implementation for clarity. The giant getter
surface on `ServerStatus` is preserved behaviorally in v2 (no semantic change);
optional future grouping is out of scope.

**Acceptance:** no exported constructor/accessor returns an interface except the
documented narrow input boundaries; `go doc` shows concrete return types;
behavior unchanged (covered by tests).

---

## 3. Connection seam + in-process mock z/OS FTP server

The blocker for unit testing today is that `Open()` dials directly — there is no
injection point. `net.Conn` is already an interface; the gap is construction.

- Add unexported seam `newSession(conn net.Conn) *FTPSession` used by `Open()`.
- Add `WithDialer(d Dialer) Option` (or `WithDialContext`) so a caller/test can
  supply the connection. `Dialer` is a tiny input interface/func type.
- Build `internal/mockzos`: a **real** in-process FTP server bound to
  `127.0.0.1:0` (ephemeral port) that speaks the z/OS FTP dialect:
  - Greeting, `USER`/`PASS`, `SYST` (returns MVS), `TYPE`, `FEAT`/`STAT`.
  - Multiline replies (`xyz-` continuation … `xyz ` terminator).
  - `PASV` → advertises a data port; accepts the data connection; streams a
    scripted payload; closes; sends `226`.
  - Scriptable per-command canned responses + scripted data payloads for
    `LIST`/`NLST`, dataset listings, `SITE`, `XSTA`, JES submit/status.
- Tests run the **real client** (`Open` → `Login` → operation) against `mockzos`,
  exercising dial, passive negotiation, data-channel transfer, and multiline
  parsing end-to-end — no mainframe, no network beyond loopback.

**Acceptance:** a test can `Open(mock.Addr())`, `Login`, run a passive LIST and a
binary + ASCII transfer fully in-process; the seam is unexported (not new public
surface beyond `WithDialer`).

---

## 4. Parsers — table-driven + real fixtures

Rewrite the dataset-listing, PDS-member, JES-spool, and LIST/NLST parsers to be
robust and declarative, consistent with the `ro-ag/parser` philosophy (layouts
described as data, decoders per field type).

- Represent each fixed-width format as a **field table**:
  `{ name, start, width (0 = to EOL), decode }`. Parse by slicing per field;
  no scattered offset arithmetic.
- **Row classifiers** handle special states before field parsing:
  - Dataset: `Migrated`, `Not Mounted`/tape, VSAM, GDG, error markers,
    `BLKSIZE` overflow indicator (the case fixed in commit `64753fc`).
  - JES: interface level 1 vs 2 (header detection), `ACTIVE` jobs, ABEND/JCL
    error, RC extraction.
- Keep the typed field wrappers (`FieldString`/`FieldInt`/…) or simplify; decide
  during implementation, preserving exported parse outputs' shape where sensible.

### Fixtures fidelity (explicit honesty rule)

- Existing `hfs/*_test.txt` are treated as **real captures = ground truth**;
  parsers must match them exactly (golden tests).
- For cases lacking a real capture, fixtures are **constructed from
  IBM-documented LIST/dataset formats and clearly labeled spec-derived**
  (header comment in the fixture file), **or** flagged in the plan for the
  maintainer to capture on a real LPAR. Synthesized output is never presented as
  a real capture.
- Add **negative/malformed** fixtures and assert specific parse errors.

**Acceptance:** every parser has golden tests against fixtures with exact-match
assertions; special-case rows each have a dedicated fixture; malformed inputs
produce specific, asserted errors; fixture provenance is labeled.

---

## 5. Unit tests — the priority

Real coverage, run with `-race`, no mainframe required. Built on §3 (mockzos) and
§4 (fixtures).

Coverage targets:
- Command/response sequences (`SendCommand`, context variant, `CheckLast`),
  including multiline replies and unexpected-code paths.
- `ReturnCode` semantics and the `ReturnError` custom error type
  (`errors.Is`/`As`, `ReturnCode()` accessor, message formatting).
- Every parser via fixtures (§4).
- Transfer: ASCII vs binary type switching + restore; EOL conversion on ASCII
  store (CRLF behavior); binary passthrough; `REST`/offset resume.
- Passive mode: `PASV` response parsing (incl. malformed), data-conn lifecycle.
- Dataset operations: `ListDatasets`, `ListPds`, `ListSpool`, `SetDataSpecs`
  (SITE command construction).

Integration tests guarded by `ZFTP_*` env vars are kept and skip cleanly when
unset (unchanged behavior), so CI never needs a real host.

**Acceptance:** `go test -race ./...` green locally and in CI; meaningful
coverage of the modules above (not token tests); no test requires a mainframe.

---

## 6. CI — GitHub Actions, pure-Go rigor

Workflow `.github/workflows/ci.yml`:

- **Triggers:** `push` + `pull_request`; `paths-ignore` for docs-only changes
  (`**/*.md`, `docs/**`).
- **Top-level** `permissions: contents: read`.
- **concurrency:** group by ref, `cancel-in-progress: true` (kill superseded PR
  runs).
- **Matrix:** `ubuntu-latest` + `macos-latest`.
- **Lanes (CGO is the subtlety):**
  - *Build/vet lane* with `CGO_ENABLED=0`: `go build ./...`, `go vet ./...`,
    `staticcheck ./...` — proves the library is genuinely pure-Go.
  - *Race lane* with `CGO_ENABLED=1`: `go test -race ./...`. The Go race detector
    requires cgo; GitHub ubuntu/macos runners ship a C compiler. This compiles
    no cgo from our source — only links the race runtime — so the pure-Go claim
    (proven by the build/vet lane) stands.
- **govulncheck** step (`golang.org/x/vuln/cmd/govulncheck`).
- All third-party actions **pinned by commit SHA** (with version comment).
- One **`CI Success`** gate job that `needs:` all matrix/lint/vuln jobs — the
  single required status check.
- **Dependabot** (`.github/dependabot.yml`): `gomod` + `github-actions`
  ecosystems, weekly, grouped updates.

**Acceptance:** CI is green on the PR; the gate job exists; actions are SHA-pinned;
Dependabot config present.

---

## 7. README

- Fix the GoDoc badge, doc link, and example import: `.v0` → `.v2`
  (current README points at `zftp.v0`).
- Add a compilable, runnable example (Open → Login → transfer/list → Close).
- Sharpen the pitch with accurate, fact-checkable claims: pure-Go, zero external
  dependencies, Apache-2.0, z/OS-specialized (datasets, JES, HFS, SITE, passive,
  ASCII/binary + EOL). No unverifiable performance or compatibility claims.
- License + (later) install-the-binary sections.

**Acceptance:** no `.v0`/`.v1` references in install/doc/example; example builds;
claims are verifiable against the code.

---

## 8. CLI-ready structure (no CLI build this effort)

Keep the library zero-dep; reserve the slot so the future CLI cannot pollute the
library's `go.mod`.

- Create `cli/` as a **separate module**: `cli/go.mod`, proposed module path
  `github.com/ro-ag/zftp/cli` (binaries ship from GitHub Releases; library keeps
  the gopkg.in import path), with a `replace` directive to the parent for local
  development.
- Minimal placeholder `main.go` (e.g., prints version) — enough to prove
  dependency isolation; not a functional CLI.
- Document the follow-up (`ZM09`): cobra CLI + GoReleaser producing multi-OS/arch
  binaries + checksums on tag, bundling `LICENSE`/`NOTICE` per Apache §4.

**Acceptance:** `cli/` builds independently; root `go.mod` remains free of CLI
deps (`go mod graph` for the library shows zero external requires).

---

## 9. Delivery & tracking (mirror `ro-ag/parser`)

- Spec: this file. Plan: `docs/superpowers/plans/2026-06-21-zftp-v2-modernization.md`.
- Feature branch `v2-modernization`, pushed; **one PR** for the full
  modernization.
- GitHub issues prefixed `ZM##`, mapping to plan phases, plus a tracking/epic
  issue linking them:
  - `ZM01` Apache-2.0 relicense + NOTICE + SPDX
  - `ZM02` v2 module bump + import rewrite
  - `ZM03` API cleanup (concrete returns, narrow input interfaces)
  - `ZM04` connection seam + `internal/mockzos`
  - `ZM05` parser rewrite (table-driven) + fixtures + golden tests
  - `ZM06` unit test suite (`-race`)
  - `ZM07` CI + Dependabot
  - `ZM08` README + GoDoc fixes + runnable example
  - `ZM09` (follow-up) CLI + GoReleaser binaries

## 10. Verification (before claiming done)

Show actual output for:
- `CGO_ENABLED=0 go build ./...`
- `CGO_ENABLED=0 go vet ./...`
- `staticcheck ./...`
- `go test -race ./...`
- `govulncheck ./...`
- CI green on the PR.

## Risks & mitigations

- **Fixture realism:** can't generate new real z/OS output without an LPAR.
  Mitigation: ground-truth existing captures, label spec-derived additions, flag
  gaps for maintainer capture (§4).
- **Hidden protocol coupling in the parser rewrite:** mitigated by golden tests
  pinning current correct behavior before refactor (test-first).
- **API rename churn:** mitigated by the hard-break decision + ~0 adoption; all
  call sites are in-repo.
