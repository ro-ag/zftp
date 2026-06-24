# zftp v2.0.0 — Release-Gate Review Findings

Date: 2026-06-24. Status: **NOT ready to tag `v2.0.0`** — blockers + majors below.
Gate baseline at review time: `go build`/`vet`/`staticcheck`/`govulncheck` clean, full
`-race` suite green. All findings below are **latent** (existing tests do not catch them).

Module path `gopkg.in/ro-ag/zftp.v2` is correct (gopkg.in `.v2` convention) — not a finding.

Items M2/M4/M5/M7 are corroborated by a deep-research pass on z/OS FTP control-reply
semantics (RFC 959 §4.2/§4.1.3/§5, IBM z/OS Comms Server; APAR PQ54076 confirms the
RST-on-abort behavior behind M7).

---

## Resolution — branch `harden/v2-readiness` (2026-06-24)

All work TDD (RED→GREEN), full gate green under `-race` (gofmt/build/vet/staticcheck/govulncheck + cli).

| Item | Status |
|------|--------|
| B1 value receivers | **Fixed** — `Field*`/`Info*` `String`/`Value`/`MarshalJSON`/`IsOverflow` now value receivers; `job --json` verified. |
| B2 `Detail() []JobDetail` | **Fixed** — `jobDetail`→`JobDetail` exported. |
| M1 `TYPE \x00` | **Fixed** — `newSession` seeds `currType=TypeAscii`. |
| M2 unbounded reads | **Fixed** — `SendCommand` bounded by reply timeout. |
| M3 TLS data handshake | **Fixed** — `tlsHandshakeBounded` with deadline. |
| M4 EOF-as-success | **Fixed** — `check` returns `io.ErrUnexpectedEOF` on EOF-before-terminator. |
| M5 `checkLast` nil-on-closed | **Fixed** — returns `net.ErrClosed`. |
| M6 JES abend | **Done** — built from real captures: real data showed abends land on `IEF450I … ABEND=Scde` (not `$HASP395`). `abendCodeRegex` matches real `Scde`/`Ucde`; `InfoJobDetail.AbendCode()` exposes the code; `ReturnCode()`→`(-1,ErrAbendedJob)`; `ErrAbend` sentinel + `classifyJesOutput` precedence; fixture corrected `ABEND=806`→`ABEND=S806`. |
| M7 passive-abort desync | **Fixed** — `checkLast` closes the session on a non-completion (426) post-transfer reply. |
| Minors | **Fixed**: doc-coverage gate extended to hfs/eol + ~48 idents documented; dead `utils` code + `StandardizeQuote` deleted; `parseCommand` trailing space; Login user hygiene; `TransferType(0).Name`→"UNKNOWN"; `eol.Cr` `// Deprecated:`; signal-handler + SITE-clobber documented. |
| API-shape | **Done**: `(sz,msg,err)` triple dropped from `*IO`→`(int64,error)`; `WithRecfm*`→`Recfm*`; `GetUser`→`User`; `ServerStatus.Snapshot()` added (one STAT round-trip; best-effort `Lines()`/`Values()` since the STAT block is prose, not key=value). **Kept**: `StatusOf`/`SetStatusOf` (38+ sites, churn > value). |
| cli `go install` | Unchanged — documented "dev builds only"; binary-first distribution. |

---

## BLOCKERS — fix before tag (permanent API lock-in / ships broken)

- [ ] **B1. Pointer-receiver `String()` + `MarshalJSON` on every `hfs.Field*`/`Info*` type, but API returns value slices.**
  Empirically proven: `json.Marshal(InfoJob value)` → `{"Name":{},...}` (non-addressable value
  cannot dispatch a pointer-receiver `MarshalJSON`); pointer and slice marshal fine.
  Symptoms: (a) `zftp job --json` ships emitting `{}` for every field
  (`cli/internal/cmd/jes.go:83`); (b) `fmt.Println(datasets[0])` never calls `String()` —
  value does not satisfy `fmt.Stringer`.
  Fix: make `String()`/`MarshalJSON` **value receivers** on
  `hfs/attributes.go` (FieldString/Int/Float/Date/Time: 28,36,97,120,176,187,232,243,276,287),
  `hfs/dataset.go:113`, `hfs/partitioned.go:27`, `hfs/spool.go:54`. One change fixes both. Breaking-OK now.

- [ ] **B2. `InfoJobDetail.Detail() []jobDetail` returns an unexported type** — `hfs/spool.go:171`.
  External callers cannot name the type → per-step JES detail permanently unusable.
  Fix: export `jobDetail` → `JobDetail`. Trivial, breaking-OK now, impossible post-tag.

---

## MAJOR — correctness / liveness (should fix before GA)

- [ ] **M1. Fresh session sends `TYPE \x00` on first transfer.** `newSession` (`ftp.go:99`) never
  initializes `currType`; `currentType()` returns `TransferType(0)` until a SetType. On
  `Open→Login→Get`, `defer restoreType(0,…)` → `SetType(0)` → `"TYPE \x00"` (`transfer.go:50`).
  z/OS rejects it; since the transfer succeeded, `restoreType` injects the error into the named
  return → a successful Get/Put returns a spurious error. Masked by a lenient mock.
  Fix: init `currType = TypeAscii` (RFC 959 default representation type) in `newSession`.

- [ ] **M2. Unbounded control reads.** `sendLocked` (`cmd.go:48`) arms a deadline only when the
  context has one; `SendCommand` passes `context.Background()`. So the `REST` reply
  (`transfer.go:117`), the `RETR/STOR` 125/150 reply (`transfer.go:122`), and every internal
  command block forever on a server that accepts then stalls. Only `checkLast` is bounded.
  Fix: route REST + the data command through a bounded context; consider a default per-command deadline.

- [ ] **M3. Data-connection TLS handshake unbounded.** `passive.go:170` `tls.Client(conn, …)` with no
  `HandshakeContext`/deadline; handshake fires lazily on first Read/Write. `DialTimeout` covered
  only the TCP dial → a stalled FTPS data negotiation hangs the transfer.
  Fix: `tls.Conn.HandshakeContext` with a deadline (or `SetDeadline` around first use).

- [ ] **M4. EOF-truncated multiline reply reported as success.** `codes.go:143` — on EOF after the
  opening line, `check` breaks; if `openingCode == expected`, the guard at `codes.go:191` is false
  → returns `(partialText, nil)`. A reply cut off after its opening line is consumed as complete
  success, leaving a desynced session for the next call. (Distinct from ZH02/#41 = EOF *before* opening.)
  Fix: track `sawTerminator`; on EOF without it return `io.ErrUnexpectedEOF` (→ session close).

- [ ] **M5. `checkLast` returns `("",nil)` on a closed session** — `cmd.go:128`. `confirmData` then
  reports an **unconfirmed transfer as success** when the session was closed concurrently. Public
  `CheckLast` also yields silent empty-success.
  Fix: return `net.ErrClosed` so completion is never claimed without a confirmed code.

- [ ] **M6. JES abends not detected.** `jes.go` `classifyJesOutput` keys on `abaMessages`
  (ABAxxx = DFSMShsm *Aggregate Backup Assist*, not abends). A real `S0C7`/`U0778` surfaces as
  `$HASP395 … ENDED - ABEND=S0C7`, matching neither the tables nor the RC regex → misleading
  "failed to retrieve job-id", no abend signal. Doc comments calling ABAxxx "abend messages" are wrong.
  Fix: add an abend matcher + sentinel; correct the comments. (Confirm against a real-LPAR job — ZH12 no-fabricate rule.)

- [ ] **M7. Passive-mode abort leaks a multi-reply control desync ("message mess").** *(added 2026-06-24)*
  Aborting an in-progress passive transfer makes z/OS emit **more than one** control reply —
  `426 Connection closed; transfer aborted.` then `226 … ABOR command successful.` — and the count
  is **non-deterministic** (if the transfer finished a beat before the abort landed, only `226`).
  Any abort path that closes **only the data connection** and keeps the control connection drains
  one reply and leaves the rest buffered → the next command reads shifted garbage. Compounded by
  z/OS **RST-on-abort** (APAR PQ54076) and the **async** `226`. Plain `ABOR\r\n` may sit queued
  behind in-flight data — RFC 959 §4.1.3 requires Telnet **IP** (`0xFF 0xF4`) + **Synch**
  (`0xFF 0xF2`, TCP urgent) so the server sees it out-of-band.
  Current state: the data-stream-error path closes the whole session (`transfer.go:132`, correct);
  there is **no** `Abort()`/keep-session path that drains correctly.
  Fix — pick one, no middle ground:
  - **A. Proper ABOR (keep session):** send Telnet IP+Synch+ABOR, then loop-read control replies
    under a deadline, discarding `426`, draining until the final `2xx`. Only then is the stream reusable.
  - **B. Abort-by-close (bulletproof):** route every abort through `s.Close()`. No drain, no reconcile.
  Forbid the "close just the data conn and keep going" path — that is exactly what produces the mess.

---

## MINOR — polish (shippable; fast-follow)

- [ ] **doc-coverage gate is half-blind** — `doc_coverage_test.go:23` scans only `os.ReadDir(".")`;
  `hfs`/`eol` exported idents ungated (~20 undocumented). Walk subpackages.
- [ ] **Dead code** — `utils.Prefix/Suffix/PrefixString/SuffixString/IsMigrated/IsNotMounted`
  (`internal/utils/utils.go:20-46`) have zero non-test callers (`Prefix/Suffix` would panic on a
  blank line if revived). Delete.
- [ ] **`StandardizeQuote` is a no-op sanitizer** — `internal/utils/utils.go:118` discards the
  `ReplaceAllString` result, quotes the unmodified name. Assign the result, or remove.
- [ ] **SITE attr leak** — `SubmitJesGetByDSN` (`jes.go:106`) saves/restores FILETYPE but leaks
  `RECFM/LRECL/BLKSIZE` into later commands. Save/restore or document.
- [ ] **`(sz, msg string, err error)` triple** on the four `*IO` methods — every caller discards `msg`
  with `_`. Non-idiomatic, locks into the signature. Consider `(int64, error)` now.
- [ ] **Status shape** — `StatusOf()` → ~72 getters, each one XSTA round-trip, no batch `Snapshot()`.
  Permanent surface; `STAT` is the only bulk escape hatch. Consider a typed snapshot now.
- [ ] **`WithRecfmFB` etc. are bare consts, not funcs** while `WithLrecl()/WithBlkSize()` are funcs —
  mis-signals callability; `SetDataSpecs` godoc references a nonexistent `Recfm(...)` ctor.
- [ ] **Trailing space** — `parseCommand` emits `"NOOP \r\n"` (`cmd.go:101`).
- [ ] **Login state hygiene** — sets `s.user` before USER/PASS succeed; failed login leaves
  `GetUser()` returning the attempted name.
- [ ] **Signal handler never released** — `ftp.go:111` `signal.Notify` with no `signal.Stop`; leaks a
  goroutine per `Open(WithSignalHandler())`. CLI single-session → acceptable; document.
- [ ] **cli `go install` broken from clean checkout** — `require v2.0.0` (untagged) + `replace =>../`.
  Documented "development builds only"; binary-first distribution → not a true blocker. For GA: tag
  the cli module + drop the replace, or soften the README.

### Nits
- `eol.Cr` deprecation not machine-readable (use `// Deprecated:` prefix).
- `GetUser()` vs noun-naming (`RemoteAddr`, etc.) — consider `User()`.
- `TransferType(0).Name()` returns "BINARY" — consider an "UNKNOWN"/default branch.
- `StatusOf()`/`SetStatusOf()` noun pairing reads oddly.

---

## Verified SOLID (re-checked; hardening held)

Concurrency (non-reentrant mutex, `sendLocked`/`*Locked` discipline, atomic `currType`/`isClosed`,
idempotent Close, lock-atomic Rename) · FILETYPE save/restore on every error path · REST/ASCII
`guardResume` at all four `*At` entries before any I/O · dataset positional parsing (bounds-guarded,
no panic on short lines) · `recFmt` getters (Lrecl=m[2]/BlockSize=m[3]/Recfm=m[1], no transposition) ·
data-stream-failure → session close (no desync) · `restoreType` error-preservation ·
`ReturnError` Is/Unwrap/CodeError · password masking in `parseCommand` ·
**multiline terminator detection** (`codes.go` opening-code anchoring — the ZH02 fix is correct).
