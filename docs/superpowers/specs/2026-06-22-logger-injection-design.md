# Logger Injection — per-session `slog` integration

- **Status:** Approved (brainstorm), pending implementation plan
- **Date:** 2026-06-22
- **Branch:** `feat/logger-injection`
- **Module:** `gopkg.in/ro-ag/zftp.v2`
- **Tracking:** to be promoted as a draft item in GitHub Project #3 (priority + In Progress on PR)

## Problem

`internal/log` is a **process-global singleton**. The public surface exposes only
`(*FTPSession).SetVerbose(LogLevel)` (a session method that mutates global state)
and the `LogLevel` constants. There is **no way to inject a logger**: output is
hardwired to `os.Stderr` via the stdlib `log.Logger`, and the internal `SetLogger`
takes a stdlib-`*log.Logger`-shaped interface (`Printf/Print/Panic/Prefix/Writer`)
that maps poorly onto structured, leveled loggers.

Users want zftp's logs to flow into their application's logging pipeline —
`log/slog`, `zap`, or `zerolog` — with proper levels and structured fields.

## Goals

- Let callers inject a logger so zftp logs route into their pipeline.
- Support `slog`, `zap`, and `zerolog` **without adding any third-party dependency
  to the core module**.
- Preserve the existing category model (`Server/Passive/Command/Debug` bitmask via
  `SetVerbose`, the ZH06 fix) and keep `LogLevel` source-compatible.
- Make logging **per-session**: no global mutable logging state.

## Non-goals

- Re-keying existing call sites into fully structured attribute sets. Existing
  preformatted messages become the slog *message*; richer attrs are a future,
  incremental improvement.
- Native (non-slog) adapter modules for zap/zerolog. They bridge via slog handlers.
- Changing transfer/protocol behavior. This is a logging-plumbing change only.

## Decisions (locked during brainstorm)

1. **slog is the bridge.** Core depends only on stdlib `log/slog`. zap plugs in via
   `go.uber.org/zap/exp/zapslog`; zerolog via a zerolog→slog handler (e.g.
   `samber/slog-zerolog`). Bridging code lives in the *user's* app, not in core.
2. **Per-session.** Each `*FTPSession` owns its own `*log.Logger` (wrapping a
   `*slog.Logger` + a category bitmask). No package-global logger state remains.
3. **Bitmask prefilter + `category` attribute.** `SetVerbose(LogLevel)` stays the
   "what to trace" prefilter. Enabled categories emit structured slog records:
   trace categories at `slog.LevelDebug` with a `category` attribute; `Warning`→
   `slog.LevelWarn`; `Error`→`slog.LevelError`.
4. **Accept `*slog.Logger`, default to `slog.Default()`.** Inject via a
   `WithLogger(*slog.Logger)` functional option (matching existing
   `WithSignalHandler`/`WithReplyTimeout` style) plus a `SetLogger` runtime setter.
   No injection ⇒ fall back to `slog.Default()`.

### Sub-decisions (confirmed: "All")

- **(a) Lazy default.** When no logger is injected, `slog.Default()` is resolved
  **at emit time**, not captured at `Open`, so a later `slog.SetDefault(...)` is
  honored.
- **(b) Drop `Fatal`/`Panic` from the logger.** slog has no Fatal/Panic. The sole
  real `log.Panicf` (`lists.go:25`, an invalid-command programmer error) becomes an
  `Error` log followed by a direct `panic(...)`. `log.Fatal` appears only in
  `example_test.go` and is removed there.
- **(c) Stable `component="zftp"` on every record.** The session's base logger is
  derived once via `injected.With(slog.String("component", "zftp"))`, so **all**
  zftp records (including warn/error) are greppable; trace lines additionally carry
  `category`.

## Architecture

`internal/log` is rewritten from a package of globals into a `Logger` **value type**.

```go
package log // internal/log

type Level uint32 // unchanged ZH06 bits: None=0, Server=1, Passive=2, Command=4, Debug=8, All=15

type Logger struct {
    base  atomic.Pointer[slog.Logger] // nil ⇒ resolve slog.Default().With(component) lazily
    level atomic.Uint32               // category bitmask
}

// New wraps l (nil ⇒ lazy slog.Default()) at the given category level. The
// returned Logger is safe for concurrent use.
func New(l *slog.Logger, lvl Level) *Logger

func (l *Logger) SetLevel(lvl Level)
func (l *Logger) Level() Level
func (l *Logger) SetSlog(s *slog.Logger) // nil ⇒ revert to lazy default
func (l *Logger) enabled(cat Level) bool // level.Load()&cat != 0
```

`*FTPSession` gains an unexported field `log *log.Logger`, set in `Open` (from
`WithLogger`, else `log.New(nil, NoLog)`). Every current `log.X(...)` call becomes
`s.log.X(...)`.

### Emission

A single internal helper builds the record with the **caller's** source location
(so `slog`'s `AddSource` points at the real call site, not the wrapper):

```go
func (l *Logger) emit(level slog.Level, msg string, attrs ...slog.Attr) {
    base := l.resolve()                 // injected logger, or slog.Default().With(component=zftp)
    if !base.Enabled(context.Background(), level) {
        return
    }
    var pcs [1]uintptr
    runtime.Callers(3, pcs[:])          // skip [Callers, emit, category method] → real call site
    r := slog.NewRecord(time.Now(), level, msg, pcs[0])
    r.AddAttrs(attrs...)
    _ = base.Handler().Handle(context.Background(), r)
}
```

> `component=zftp` is applied **once** at injection: `New`/`SetSlog` store
> `l.With(slog.String("component","zftp"))`. `resolve()` returns that stored base
> when set; otherwise it returns `slog.Default().With(slog.String("component",
> "zftp"))` (lazy, per (a)+(c)). Either way every emitted record is tagged, with no
> per-emit `With` on the injected path.

### Category methods (keep printf-style to minimize call-site churn)

```go
func (l *Logger) Commandf(format string, a ...any) {
    if l.enabled(CommandLevel) {
        l.emit(slog.LevelDebug, fmt.Sprintf(format, a...), slog.String("category", "command"))
    }
}
// Serverf/Passivef/Debugf — same shape, category "server"/"passive"/"debug"
func (l *Logger) Warningf(format string, a ...any) { l.emit(slog.LevelWarn, fmt.Sprintf(format, a...)) }
func (l *Logger) Errorf(format string, a ...any)   { l.emit(slog.LevelError, fmt.Sprintf(format, a...)) }
// Non-f variants (Debug/Error/Warning/...) wrap fmt.Sprint(a...) — current call sites use both.
```

### Level mapping

| zftp call | gated by | slog level | attrs |
|---|---|---|---|
| `Commandf`/`Command` | `CommandLevel` bit | `LevelDebug` | `component=zftp`, `category=command` |
| `Serverf`/`Server` | `ServerLevel` bit | `LevelDebug` | `component=zftp`, `category=server` |
| `Passivef`/`Passive` | `PassiveLevel` bit | `LevelDebug` | `component=zftp`, `category=passive` |
| `Debugf`/`Debug` | `DebugLevel` bit | `LevelDebug` | `component=zftp`, `category=debug` |
| `Warningf`/`Warning` | always | `LevelWarn` | `component=zftp` |
| `Errorf`/`Error` | always | `LevelError` | `component=zftp` |

Two filters compose: zftp's bitmask decides whether a line is emitted at all; the
slog handler's own level/handler decides whether it is recorded/printed.

## Call-site threading

~60 sites across 9 files (see appendix). Threading rules:

- **`*FTPSession` methods** (cmd.go, put.go, get.go, ftp.go, most of lists.go):
  `log.X` → `s.log.X`. Mechanical.
- **Types that log without a session:**
  - `childConnection` (passive.go) carries an unexported `lg *log.Logger`, set by
    the session when the child is created. Its methods log via `c.lg.X`.
  - `ReturnCode.check` (codes.go) gains a `*log.Logger` parameter, supplied by the
    session command path that invokes it.
- **Free helpers in `internal/utils`** (`utils.go:101`, `:188`) take a
  `*log.Logger` parameter from their session callers.
- **`internal/transfer`**: audit during implementation; if any site logs, it
  receives a `*log.Logger` parameter. (Current grep shows none.)
- **`lists.go:25` `log.Panicf`** → `s.log.Errorf(...)` then `panic(fmt.Sprintf(...))`.

Exact signatures are finalized in the implementation plan; the mechanism (pass the
session's `*log.Logger`, or store a reference on the logging type at construction)
is fixed here.

## Public API (zftp package)

```go
// WithLogger routes this session's logs into l. A nil l selects slog.Default().
func WithLogger(l *slog.Logger) Option

// SetLogger swaps the session's logger at runtime. A nil l reverts to slog.Default().
func (s *FTPSession) SetLogger(l *slog.Logger)

// SetVerbose selects which trace categories this session emits (unchanged signature).
func (s *FTPSession) SetVerbose(level LogLevel)

// LogLevel + NoLog/LogServer/LogPassive/LogCommand/LogDebug/LogAll — unchanged.
```

Default session (`Open` with no `WithLogger`, no `SetVerbose`): logger resolves to
`slog.Default()`, level `NoLog` ⇒ **silent until `SetVerbose`** (preserves today's
default-quiet behavior).

### Bridging examples (documentation)

```go
// slog (stdlib) — direct
h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
s, _ := zftp.Open(addr, zftp.WithLogger(slog.New(h)))
s.SetVerbose(zftp.LogCommand | zftp.LogServer)

// zap — via a zap→slog handler (go.uber.org/zap/exp/zapslog)
s, _ := zftp.Open(addr, zftp.WithLogger(slog.New(zapHandler)))   // zapHandler is a slog.Handler

// zerolog — via a zerolog→slog handler (e.g. samber/slog-zerolog)
s, _ := zftp.Open(addr, zftp.WithLogger(slog.New(zerologHandler))) // zerologHandler is a slog.Handler
```

*Exact handler constructors are taken from each bridge's own docs (verify against
the current release before pasting into the README).* The bridge imports live in the
**user's** go.mod; core stays zero-dep — all zftp needs is any `slog.Handler`.

## Error handling / concurrency

- A `nil` logger is always valid input (⇒ lazy `slog.Default()`); no nil panics.
- `level` is an `atomic.Uint32`; the base `*slog.Logger` is an
  `atomic.Pointer[slog.Logger]`. Reads are lock-free; concurrent `SetLogger` /
  `SetVerbose` / emit are race-free. Verified under `-race`.
- Handler errors from `Handle` are ignored (slog convention; nothing actionable).

## Testing (TDD, RED→GREEN, `-race`)

A reusable **capturing `slog.Handler`** (records `[]slog.Record` + attrs) backs the
assertions.

**internal/log (white-box, extend `logger_test.go`):**

- Prefilter: with level `CommandLevel`, `Commandf` emits exactly one record; the
  other three trace categories emit zero.
- Level mapping: trace categories ⇒ `LevelDebug`; `Warningf`⇒`LevelWarn`;
  `Errorf`⇒`LevelError`.
- Attrs: every record carries `component=zftp`; trace records carry the right
  `category`.
- Always-on: `Warningf`/`Errorf` emit even at level `None`.
- Default resolution: `New(nil, …)` and `SetSlog(nil)` route to `slog.Default()`
  (swap the default with a capturing handler via `slog.SetDefault`, restore in
  `t.Cleanup`); lazy — a `SetDefault` after construction is honored.
- Source: emitted record `PC` resolves to the test's call site (not the wrapper).
- Concurrency: goroutines hammering `SetSlog`/`SetLevel`/`Commandf` under `-race`.
- ZH06 invariants preserved (distinct bits, isolation) — keep existing cases.

**zftp (black-box via `internal/mockzos`):**

- `WithLogger` wires the session: run a mock command flow, assert the injected
  handler captured `category=command`/`server` records.
- **Per-session isolation (the core win):** two sessions with different injected
  handlers and different `SetVerbose` levels — each captures only its own records;
  no cross-talk.
- Back-compat: a default session (no `WithLogger`, no `SetVerbose`) captures zero
  records.

## Breaking changes & migration (pre-ship, approved)

- `go.mod`: `go 1.20` → **`go 1.21`** (slog is stdlib from 1.21).
- CI matrix `.github/workflows`: `["1.20","stable"]` → `["1.21","stable"]`.
- Removed (internal only): package-global log funcs, the stdlib-shaped `Logger`
  interface, `internal/log.SetLogger/SetOutput/SetLevel` globals.
- Public: `SetVerbose`/`LogLevel` source-compatible; **new** `WithLogger` +
  `SetLogger`. Log *output format* changes (now slog records).
- v2 is unreleased ⇒ breaking changes are approved.

## File-by-file change list

- `internal/log/logger.go` — rewrite to `Logger` struct + slog emission.
- `internal/log/logger_test.go` — extend (capturing handler, mapping, default, race).
- `log.go` — keep `LogLevel`; (option may live here or in the options file).
- `ftp.go` — `*FTPSession.log` field; `Open` wiring; `WithLogger`; `SetLogger`;
  `SetVerbose` → `s.log.SetLevel`.
- `cmd.go`, `codes.go`, `passive.go`, `put.go`, `get.go`, `lists.go`, `transfer.go`
  — `log.X` → session/childConn/param logger; `Panicf`→`Error`+`panic`.
- `internal/utils/utils.go`, `internal/transfer/*` — logging helpers take
  `*log.Logger`.
- `go.mod`, `.github/workflows/*.yml` — version bump.
- `README` / examples — logger injection + zap/zerolog bridge snippets.

## Risks / open questions

- **Source line (`AddSource`)**: the `runtime.Callers` skip depth must be verified
  empirically (Sprintf vs non-f paths may differ); a test pins it.
- **`internal/transfer` logging**: confirm there are truly no sites; adjust the
  threading list if the audit finds any.
- **Default `component` cost**: deriving `slog.Default().With(...)` lazily per emit
  allocates; if it shows up in profiles, cache the derived default and invalidate on
  `slog.SetDefault` (acceptable to defer — tracing is opt-in/off by default).

## Acceptance criteria

- Injecting a `*slog.Logger` routes zftp logs into it; zap/zerolog work via their
  slog handlers with **zero** new core dependencies.
- Logging is per-session: two sessions log independently (no global state); proven
  by an isolation test.
- `SetVerbose`/`LogLevel` unchanged for callers; ZH06 bit invariants intact.
- `CGO_ENABLED=0 go build ./...`, `go vet ./...`, `staticcheck ./...`,
  `go test -race ./...`, `govulncheck ./...` all green; CI green on `go 1.21` +
  `stable`.
