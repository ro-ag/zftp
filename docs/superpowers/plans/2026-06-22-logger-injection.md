# Logger Injection (per-session slog) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let callers inject a `*slog.Logger` per `FTPSession` so zftp logs route into slog/zap/zerolog, with zero third-party deps in core.

**Architecture:** Rewrite `internal/log` from a global singleton into a `Logger` value type that applies the category bitmask prefilter and emits structured `slog` records. Each `*FTPSession` owns one. Transitional package-level shims keep the build green while ~60 call sites migrate file-by-file; the shims are deleted at the end.

**Tech Stack:** Go 1.26 (`log/slog`; floor set to current Go — see Global Constraints), `internal/mockzos` loopback for tests.

## Global Constraints

- **Go floor:** `go 1.26` in `go.mod` + `cli/go.mod`; CI matrix `["1.26","stable"]`. (slog needs ≥1.21; floor set to 1.26 to dodge the Go 1.21 macOS `-race` LC_UUID toolchain bug.)
- **Zero third-party deps in core.** Only stdlib (incl. `log/slog`). zap/zerolog bridge in the user's app.
- **SPDX header** `// SPDX-License-Identifier: Apache-2.0` on every `.go` file (first line).
- **Errors:** `errors.New` for static sentinels, `%w` for wraps.
- **Commits/PR:** NO `Co-Authored-By` and NO "Generated with Claude"/AI attribution. Branch `feat/logger-injection` off `main`; one PR.
- **Gate before done (show output):** `CGO_ENABLED=0 go build ./...`, `CGO_ENABLED=0 go vet ./...`, `staticcheck ./...`, `go test -race ./...`, `govulncheck ./...`; CI green.
- **ZH06 bit values are load-bearing:** `None=0, ServerLevel=1, PassiveLevel=2, CommandLevel=4, DebugLevel=8, All=15`. Public `LogLevel` must keep identical values (`SetVerbose` casts `log.Level(level)`).

---

### Task 1: Bump Go floor to 1.21

**Files:**
- Modify: `go.mod:3`
- Modify: `.github/workflows/*.yml` (the matrix line `go: ["1.20", "stable"]`)

**Interfaces:**
- Consumes: nothing.
- Produces: `log/slog` available to all later tasks.

- [ ] **Step 1: Edit go.mod**

Change `go 1.20` → `go 1.21`.

- [ ] **Step 2: Edit CI matrix**

In the workflow file with `go: ["1.20", "stable"]`, change to `go: ["1.21", "stable"]`.

- [ ] **Step 3: Verify build**

Run: `CGO_ENABLED=0 go build ./...`
Expected: success, no output.

- [ ] **Step 4: Commit**

```bash
git add go.mod .github/workflows
git commit -m "build: require Go 1.21 (log/slog)"
```

---

### Task 2: Rewrite internal/log as a Logger value type (+ transitional shims)

Rewrite the package: a `Logger` struct emitting `slog` records, reusing the ZH06 `Level` bits, plus package-level shims (same names/signatures as today) so existing call sites compile unchanged. TDD the struct.

**Files:**
- Modify (full rewrite): `internal/log/logger.go`
- Modify (extend): `internal/log/logger_test.go`

**Interfaces:**
- Consumes: `log/slog` (Task 1).
- Produces:
  - `type Level uint32` with `None=0, ServerLevel=1, PassiveLevel=2, CommandLevel=4, DebugLevel=8, All=15`.
  - `type Logger struct{…}`; `func New(l *slog.Logger, lvl Level) *Logger`.
  - Methods: `SetSlog(*slog.Logger)`, `SetLevel(Level)`, `Level() Level`, and emitters `Command/Commandf/Server/Serverf/Passive/Passivef/Debug/Debugf/Warning/Warningf/Error/Errorf` (each `(v ...any)` / `(format string, v ...any)`).
  - Transitional package funcs (same names) delegating to a package default `std *Logger`: `SetLevel`, `Debug/Debugf/Command/Commandf/Passive/Passivef/Server/Serverf/Warning/Warningf/Error/Errorf`.

- [ ] **Step 1: Write the failing tests** (extend `internal/log/logger_test.go`)

Add a capturing handler and behavior tests. KEEP the existing ZH06 tests (`TestLevelsAreDistinctSingleBits`, etc.) unchanged.

```go
// capture is a slog.Handler that records every record it is asked to handle.
type capture struct {
	mu      sync.Mutex
	records []slog.Record
}

func (c *capture) Enabled(context.Context, slog.Level) bool { return true }
func (c *capture) Handle(_ context.Context, r slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = append(c.records, r.Clone())
	return nil
}
func (c *capture) WithAttrs(attrs []slog.Attr) slog.Handler { return c }
func (c *capture) WithGroup(string) slog.Handler            { return c }

func (c *capture) snapshot() []slog.Record {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]slog.Record(nil), c.records...)
}

// attrMap collects a record's attributes into a map for assertions.
func attrMap(r slog.Record) map[string]string {
	m := map[string]string{}
	r.Attrs(func(a slog.Attr) bool { m[a.Key] = a.Value.String(); return true })
	return m
}

func TestLogger_PrefilterEmitsOnlyEnabledCategory(t *testing.T) {
	cap := &capture{}
	lg := New(slog.New(cap), CommandLevel)

	lg.Commandf("PASS %s", "secret")
	lg.Serverf("220 ready")  // ServerLevel not enabled
	lg.Passivef("227 ...")   // PassiveLevel not enabled
	lg.Debugf("trace")       // DebugLevel not enabled

	recs := cap.snapshot()
	if len(recs) != 1 {
		t.Fatalf("want exactly 1 record (command only), got %d", len(recs))
	}
	if recs[0].Level != slog.LevelDebug {
		t.Errorf("command should map to LevelDebug, got %v", recs[0].Level)
	}
	m := attrMap(recs[0])
	if m["category"] != "command" {
		t.Errorf(`want category="command", got %q`, m["category"])
	}
	if m["component"] != "zftp" {
		t.Errorf(`want component="zftp", got %q`, m["component"])
	}
	if recs[0].Message != "PASS secret" {
		t.Errorf("unexpected message %q", recs[0].Message)
	}
}

func TestLogger_WarningAndErrorAlwaysEmit(t *testing.T) {
	cap := &capture{}
	lg := New(slog.New(cap), None) // nothing in the trace bitmask

	lg.Warningf("w")
	lg.Errorf("e")

	recs := cap.snapshot()
	if len(recs) != 2 {
		t.Fatalf("warning+error must emit regardless of level, got %d", len(recs))
	}
	if recs[0].Level != slog.LevelWarn || recs[1].Level != slog.LevelError {
		t.Errorf("levels: got %v, %v want Warn, Error", recs[0].Level, recs[1].Level)
	}
	if attrMap(recs[0])["component"] != "zftp" {
		t.Error("warning record missing component=zftp")
	}
}

func TestLogger_SourcePointsAtCallSite(t *testing.T) {
	cap := &capture{}
	lg := New(slog.New(cap), DebugLevel)

	lg.Debugf("x") ; wantLine := 0 // placeholder; set below
	_ = wantLine

	recs := cap.snapshot()
	if len(recs) != 1 {
		t.Fatalf("want 1 record, got %d", len(recs))
	}
	fs := runtime.CallersFrames([]uintptr{recs[0].PC})
	f, _ := fs.Next()
	if !strings.HasSuffix(f.File, "logger_test.go") {
		t.Errorf("source should be the call site (logger_test.go), got %s:%d", f.File, f.Line)
	}
}

func TestLogger_NilSlogUsesDefault(t *testing.T) {
	cap := &capture{}
	prev := slog.Default()
	slog.SetDefault(slog.New(cap))
	t.Cleanup(func() { slog.SetDefault(prev) })

	lg := New(nil, DebugLevel) // nil ⇒ lazy slog.Default()
	lg.Debugf("via default")

	if len(cap.snapshot()) != 1 {
		t.Fatal("nil logger should route to slog.Default()")
	}
}

func TestLogger_ConcurrentSwapRace(t *testing.T) {
	lg := New(slog.New(&capture{}), All)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				lg.Commandf("c%d", j)
				lg.SetLevel(All)
				lg.SetSlog(slog.New(&capture{}))
			}
		}()
	}
	wg.Wait()
}
```

Add imports: `context`, `log/slog`, `runtime`, `strings`, `sync` (plus existing `math/bits`, `testing`).

> Note for the `TestLogger_SourcePointsAtCallSite` body: delete the placeholder
> lines; just call `lg.Debugf("x")` then assert. (Shown verbosely to make the
> intent explicit.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/log/ -run 'TestLogger_' -v`
Expected: FAIL — `New`, `*Logger` methods undefined.

- [ ] **Step 3: Write the implementation** (full file replace `internal/log/logger.go`)

```go
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync/atomic"
	"time"
)

// Level is a bitmask of independent logging categories; combine with OR and test
// membership via the category emitters. Each flag is a distinct bit, so enabling
// one category never implies another.
type Level uint32

// None disables every logging category.
const None Level = 0

const (
	ServerLevel  Level = 1 << iota // 1 — server replies
	PassiveLevel                   // 2 — passive-mode negotiation
	CommandLevel                   // 4 — commands sent
	DebugLevel                     // 8 — verbose debug
)

// All enables every logging category.
const All = ServerLevel | PassiveLevel | CommandLevel | DebugLevel

// component tags every zftp record so it is greppable in a shared log stream.
const component = "zftp"

var (
	catCommand = slog.String("category", "command")
	catServer  = slog.String("category", "server")
	catPassive = slog.String("category", "passive")
	catDebug   = slog.String("category", "debug")
)

// Logger applies the category bitmask prefilter and emits structured slog records.
// The zero value is not usable; construct with New. Safe for concurrent use.
type Logger struct {
	base  atomic.Pointer[slog.Logger] // nil ⇒ resolve slog.Default() lazily at emit
	level atomic.Uint32               // category bitmask
}

// New returns a Logger emitting to l at the given category level. A nil l selects
// slog.Default() at emit time (honoring a later slog.SetDefault).
func New(l *slog.Logger, lvl Level) *Logger {
	lg := &Logger{}
	lg.SetSlog(l)
	lg.SetLevel(lvl)
	return lg
}

// SetSlog swaps the destination logger. A nil s reverts to the lazy slog.Default().
func (l *Logger) SetSlog(s *slog.Logger) {
	if s == nil {
		l.base.Store(nil)
		return
	}
	l.base.Store(tagged(s))
}

// SetLevel replaces the category bitmask.
func (l *Logger) SetLevel(lvl Level) { l.level.Store(uint32(lvl)) }

// Level returns the current category bitmask.
func (l *Logger) Level() Level { return Level(l.level.Load()) }

func (l *Logger) enabled(cat Level) bool { return l.level.Load()&uint32(cat) != 0 }

// tagged derives a logger carrying the stable component attribute (applied once).
func tagged(s *slog.Logger) *slog.Logger {
	return s.With(slog.String("component", component))
}

// resolve returns the destination logger, tagging the lazy default when no logger
// was injected.
func (l *Logger) resolve() *slog.Logger {
	if b := l.base.Load(); b != nil {
		return b
	}
	return tagged(slog.Default())
}

// emit builds a record with the caller's source location and dispatches it.
func (l *Logger) emit(level slog.Level, msg string, attrs ...slog.Attr) {
	base := l.resolve()
	ctx := context.Background()
	if !base.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip [Callers, emit, category method] → real call site
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.AddAttrs(attrs...)
	_ = base.Handler().Handle(ctx, r)
}

func (l *Logger) Command(v ...any)            { if l.enabled(CommandLevel) { l.emit(slog.LevelDebug, fmt.Sprint(v...), catCommand) } }
func (l *Logger) Commandf(f string, v ...any) { if l.enabled(CommandLevel) { l.emit(slog.LevelDebug, fmt.Sprintf(f, v...), catCommand) } }
func (l *Logger) Server(v ...any)             { if l.enabled(ServerLevel) { l.emit(slog.LevelDebug, fmt.Sprint(v...), catServer) } }
func (l *Logger) Serverf(f string, v ...any)  { if l.enabled(ServerLevel) { l.emit(slog.LevelDebug, fmt.Sprintf(f, v...), catServer) } }
func (l *Logger) Passive(v ...any)            { if l.enabled(PassiveLevel) { l.emit(slog.LevelDebug, fmt.Sprint(v...), catPassive) } }
func (l *Logger) Passivef(f string, v ...any) { if l.enabled(PassiveLevel) { l.emit(slog.LevelDebug, fmt.Sprintf(f, v...), catPassive) } }
func (l *Logger) Debug(v ...any)              { if l.enabled(DebugLevel) { l.emit(slog.LevelDebug, fmt.Sprint(v...), catDebug) } }
func (l *Logger) Debugf(f string, v ...any)   { if l.enabled(DebugLevel) { l.emit(slog.LevelDebug, fmt.Sprintf(f, v...), catDebug) } }
func (l *Logger) Warning(v ...any)            { l.emit(slog.LevelWarn, fmt.Sprint(v...)) }
func (l *Logger) Warningf(f string, v ...any) { l.emit(slog.LevelWarn, fmt.Sprintf(f, v...)) }
func (l *Logger) Error(v ...any)              { l.emit(slog.LevelError, fmt.Sprint(v...)) }
func (l *Logger) Errorf(f string, v ...any)   { l.emit(slog.LevelError, fmt.Sprintf(f, v...)) }

// --- transitional package-level shims (removed in the final task once every call
// site uses a *Logger). They delegate to a package default so existing call sites
// compile unchanged during the migration. ---

var std = New(nil, None)

func SetLevel(lvl Level)          { std.SetLevel(lvl) }
func Debug(v ...any)              { std.Debug(v...) }
func Debugf(f string, v ...any)   { std.Debugf(f, v...) }
func Command(v ...any)            { std.Command(v...) }
func Commandf(f string, v ...any) { std.Commandf(f, v...) }
func Passive(v ...any)            { std.Passive(v...) }
func Passivef(f string, v ...any) { std.Passivef(f, v...) }
func Server(v ...any)             { std.Server(v...) }
func Serverf(f string, v ...any)  { std.Serverf(f, v...) }
func Warning(v ...any)            { std.Warning(v...) }
func Warningf(f string, v ...any) { std.Warningf(f, v...) }
func Error(v ...any)              { std.Error(v...) }
func Errorf(f string, v ...any)   { std.Errorf(f, v...) }
```

> The shims add one stack frame, so during migration a shim-routed record's source
> points at the shim. This is transitional; direct `*Logger` calls (tests + migrated
> sites) have correct source. The source test calls methods directly, so it passes now.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -race ./internal/log/ -v`
Expected: PASS (new `TestLogger_*` + existing ZH06 tests).

- [ ] **Step 5: Verify the whole module still builds (shims keep call sites green)**

Run: `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./...`
Expected: success.

- [ ] **Step 6: Commit**

```bash
git add internal/log/logger.go internal/log/logger_test.go
git commit -m "feat(log): slog-backed per-instance Logger type (shims keep call sites green)"
```

---

### Task 3: Wire a per-session logger + public WithLogger/SetLogger

**Files:**
- Modify: `options.go` (add `logger` to `dialOptions`; add `WithLogger`)
- Modify: `ftp.go` (add `log` field; set it in `newSession`; `SetVerbose`→`s.log`; add `SetLogger`)
- Modify: `log.go` (no value change; LogLevel stays — confirm it still casts)
- Test: `log_test.go` (extend) + new `logger_session_test.go`

**Interfaces:**
- Consumes: `log.New`, `*log.Logger` methods (Task 2).
- Produces:
  - `func WithLogger(l *slog.Logger) Option`
  - `(*FTPSession).SetLogger(l *slog.Logger)`
  - `FTPSession.log *log.Logger` (unexported), always non-nil after `newSession`.

- [ ] **Step 1: Write failing tests** (`logger_session_test.go`, package `zftp`)

```go
// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	ilog "gopkg.in/ro-ag/zftp.v2/internal/log"
)

type capHandler struct {
	mu   sync.Mutex
	cats []string
}

func (c *capHandler) Enabled(context.Context, slog.Level) bool { return true }
func (c *capHandler) Handle(_ context.Context, r slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "category" {
			c.cats = append(c.cats, a.Value.String())
		}
		return true
	})
	return nil
}
func (c *capHandler) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c *capHandler) WithGroup(string) slog.Handler      { return c }
func (c *capHandler) count() int { c.mu.Lock(); defer c.mu.Unlock(); return len(c.cats) }

// Two sessions, different loggers and levels, must not cross-talk.
func TestPerSessionLoggerIsolation(t *testing.T) {
	a, b := &capHandler{}, &capHandler{}

	s1 := newSession(nil, dialOptions{})
	s1.SetLogger(slog.New(a))
	s1.SetVerbose(LogCommand)

	s2 := newSession(nil, dialOptions{})
	s2.SetLogger(slog.New(b))
	s2.SetVerbose(NoLog)

	s1.log.Commandf("PASS x")
	s2.log.Commandf("PASS y") // s2 has NoLog ⇒ suppressed

	if a.count() != 1 {
		t.Errorf("session 1 should have 1 command record, got %d", a.count())
	}
	if b.count() != 0 {
		t.Errorf("session 2 (NoLog) should capture nothing, got %d", b.count())
	}
}

// WithLogger wires the option through Open's construction path (newSession).
func TestWithLoggerOption(t *testing.T) {
	h := &capHandler{}
	var cfg dialOptions
	cfg.apply([]Option{WithLogger(slog.New(h))})
	s := newSession(nil, cfg)
	s.SetVerbose(LogServer)

	s.log.Serverf("220 ready")
	if h.count() != 1 {
		t.Errorf("WithLogger should route Serverf to the handler, got %d", h.count())
	}
}

// Default session (no WithLogger, no SetVerbose) is silent.
func TestDefaultSessionSilent(t *testing.T) {
	s := newSession(nil, dialOptions{})
	if s.log == nil {
		t.Fatal("session logger must be non-nil after newSession")
	}
	if s.log.Level() != ilog.None {
		t.Errorf("default level should be None, got %d", s.log.Level())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test . -run 'TestPerSessionLoggerIsolation|TestWithLoggerOption|TestDefaultSessionSilent' -v`
Expected: FAIL — `SetLogger`, `WithLogger`, `s.log` undefined.

- [ ] **Step 3: Implement**

In `options.go`: add field + import `log/slog` + the internal log import, and the option.

```go
// dialOptions struct: add field
	logger *slog.Logger
```
```go
// WithLogger routes this session's logs into l. A nil l selects slog.Default().
func WithLogger(l *slog.Logger) Option {
	return func(o *dialOptions) { o.logger = l }
}
```

In `ftp.go`:
- Add field to `FTPSession`:
```go
	log *log.Logger
```
- In `newSession`, set it (nil cfg.logger ⇒ lazy default):
```go
	return &FTPSession{
		conn:      conn,
		rawConn:   conn,
		reader:    bufio.NewReader(conn),
		dialCfg:   cfg,
		jobPrefix: regexp.MustCompile(`(JOB\d{5})`),
		log:       log.New(cfg.logger, log.None),
	}
```
- Replace `SetVerbose` body:
```go
func (s *FTPSession) SetVerbose(level LogLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.log.SetLevel(log.Level(level))
}
```
- Add setter:
```go
// SetLogger swaps the destination logger for this session's logs at runtime.
// A nil l reverts to slog.Default().
func (s *FTPSession) SetLogger(l *slog.Logger) {
	s.log.SetSlog(l)
}
```

> `newSession(nil, …)` is valid: the tests pass a nil conn and never do I/O —
> `bufio.NewReader(nil)` is fine when unread. Real sessions pass a live conn.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -race . -run 'TestPerSessionLoggerIsolation|TestWithLoggerOption|TestDefaultSessionSilent' -v`
Expected: PASS.

- [ ] **Step 5: Full build/vet (call sites still on shims)**

Run: `CGO_ENABLED=0 go build ./... && go test -race ./... 2>&1 | tail -15`
Expected: success; existing suite green.

- [ ] **Step 6: Commit**

```bash
git add options.go ftp.go log.go logger_session_test.go
git commit -m "feat: per-session slog logger (WithLogger option + SetLogger), SetVerbose now per-session"
```

---

### Tasks 4–8: Migrate call sites from package shims to the session logger

**Transformation rule:** replace `log.X(...)` with the session's logger. The receiver is `s.log` in `*FTPSession` methods. For non-session sites, thread a `*log.Logger` (see each task). After each file, no `log.<emitter>(` package calls remain in it. The `log` import stays only where the `log.Level`/type is still referenced (none after migration) — goimports will drop unused imports.

**Verification after every task:**
```bash
CGO_ENABLED=0 go build ./... && go vet ./... && go test -race ./... 2>&1 | tail -8
```
All green (shims still exist for not-yet-migrated files).

---

### Task 4: Migrate ftp.go + cmd.go (pure `*FTPSession` methods)

**Files:** Modify `ftp.go`, `cmd.go`.

- [ ] **Step 1: ftp.go** — these are inside `*FTPSession` methods or `Open` (which holds `session`):
  - `ftp.go:60` `log.Debug(...)` → `session.log.Debug(...)` (inside `Open`, var is `session`).
  - `ftp.go:114` `log.Errorf(...)` → `s.log.Errorf(...)` (inside `installSignalHandler`, receiver `s`).
  - `ftp.go:195` `log.Debugf(...)`, `:197` `log.Warningf(...)`, `:202` `log.Debugf(...)`, `:204` `log.Warningf(...)` → `s.log.…` (Close path; confirm receiver name).
- [ ] **Step 2: cmd.go** — all inside `*FTPSession` methods (receiver `s`):
  - `:57` `log.Commandf`, `:64` `log.Serverf`, `:94/:96/:98` `log.Commandf`, `:124` `log.Warningf`, `:144` `log.Serverf` → `s.log.…`.
- [ ] **Step 3: Confirm zero residual + verify**

Run: `rg -n 'log\.(Debug|Command|Server|Passive|Warning|Error)' ftp.go cmd.go` → no matches.
Run the verification block. Expected: green.
- [ ] **Step 4: Commit** `refactor(log): route ftp.go + cmd.go through the session logger`

---

### Task 5: Migrate put.go + get.go + transfer.go (`*FTPSession` methods)

**Files:** Modify `put.go`, `get.go`, `transfer.go`.

- [ ] **Step 1:** `put.go` (14×`Debugf`, 2×`Errorf`, 2×`Error`, 2×`Debug`, 1×`Warning` at the lines from the appendix) → `s.log.…`. Confirm each enclosing method's receiver name (`s`).
- [ ] **Step 2:** `get.go` (5×`Debug`, 4×`Debugf`) → `s.log.…`.
- [ ] **Step 3:** `transfer.go:112` `log.Error(err)` → `s.log.Error(err)` (confirm receiver; if the function is not a method, see Task 8 pattern and pass a `*log.Logger`).
- [ ] **Step 4:** `rg -n 'log\.(Debug|Command|Server|Passive|Warning|Error)' put.go get.go transfer.go` → none. Verify block. Commit `refactor(log): route put/get/transfer through the session logger`.

---

### Task 6: Migrate codes.go (`ReturnCode.check` gains a logger param)

`ReturnCode.check` logs but is not a session method. Thread a `*log.Logger`.

**Files:** Modify `codes.go`, and every `.check(` caller. Test: `codes_test.go`.

- [ ] **Step 1:** Change signature `func (rc ReturnCode) check(r *bufio.Reader) (string, error)` → `func (rc ReturnCode) check(r *bufio.Reader, lg *log.Logger) (string, error)`; inside, `log.Serverf`→`lg.Serverf`, `log.Errorf`→`lg.Errorf`.
- [ ] **Step 2:** Update callers:
  - `ftp.go:55` `CodeSvcReadySoon.check(session.reader)` → `…check(session.reader, session.log)`.
  - cmd.go / wherever `.check(` is called in the send path → pass `s.log`. Find them: `rg -n '\.check\(' --type go | rg -v _test.go`.
  - `codes_test.go` callers → pass `ilog.New(nil, ilog.None)` (import internal/log as `ilog`) or a capturing logger.
- [ ] **Step 3:** `rg -n 'log\.(Server|Error)' codes.go` → none. Verify block (incl. `go test -race .`). Commit `refactor(log): thread session logger into ReturnCode.check`.

---

### Task 7: Migrate passive.go (childConnection carries a logger) + lists.go

**Files:** Modify `passive.go`, `lists.go`.

- [ ] **Step 1:** Add an unexported field to `childConnection`: `lg *log.Logger`. Set it where the child is created (the session creating it passes `s.log`). Replace the 6 `log.Debugf` in `passive.go` with `c.lg.Debugf(...)` (confirm receiver name on each method).
- [ ] **Step 2:** `lists.go`:
  - `:36`, `:52` `log.Error(err)` → `s.log.Error(err)`; `:69` `log.Passivef` → `s.log.Passivef` (confirm `anyList` has the session; if it takes `s *FTPSession`, use `s.log`).
  - `:25` `log.Panicf("invalid command: %s", cmd)` → 
    ```go
    s.log.Errorf("invalid command: %s", cmd)
    panic(fmt.Sprintf("invalid command: %s", cmd))
    ```
    (add `fmt` import if needed). If no session is in scope at that point, keep a bare `panic(fmt.Sprintf(...))` — it is a programmer-error guard.
- [ ] **Step 3:** `rg -n 'log\.(Debug|Passive|Error|Panic)' passive.go lists.go` → none. Verify block. Commit `refactor(log): childConnection logger + lists.go (drop log.Panicf)`.

---

### Task 8: Migrate internal/utils (logging helpers take a `*log.Logger`)

**Files:** Modify `internal/utils/utils.go` and its callers.

- [ ] **Step 1:** For each helper that logs (`utils.go:101` `Debugf`, `:188` `Warning`), add a trailing `lg *log.Logger` parameter and replace `log.X`→`lg.X`. (Import `gopkg.in/ro-ag/zftp.v2/internal/log`.)
- [ ] **Step 2:** Update callers (in the `zftp` package) to pass `s.log`. Find them by the helper names: `rg -n '<helperName>\(' --type go`.
- [ ] **Step 3:** Audit `internal/transfer`: `rg -n 'internal/log|log\.' internal/transfer`. If any site logs, apply the same `*log.Logger`-param pattern; else no change.
- [ ] **Step 4:** `rg -rn 'log\.(Debug|Command|Server|Passive|Warning|Error)' internal/utils internal/transfer` → none. Verify block. Commit `refactor(log): thread logger into internal/utils helpers`.

---

### Task 9: Remove transitional shims; confirm clean

**Files:** Modify `internal/log/logger.go`.

- [ ] **Step 1:** Confirm no remaining package-shim users:
  Run: `rg -n 'log\.(SetLevel|Debug|Debugf|Command|Commandf|Passive|Passivef|Server|Serverf|Warning|Warningf|Error|Errorf)\(' --type go | rg -v 'internal/log/|\.log\.|_test.go'`
  Expected: no matches. (Anything left is an un-migrated site — migrate it before deleting shims.)
- [ ] **Step 2:** Delete the shim block (the `var std` + the package funcs) from `internal/log/logger.go`.
- [ ] **Step 3:** Verify the source-line test still passes (records now never go through a shim):
  Run: `go test -race ./internal/log/ -run TestLogger_SourcePointsAtCallSite -v` → PASS.
- [ ] **Step 4:** Full verify block + `go test -race ./...`. Commit `refactor(log): remove transitional package shims`.

---

### Task 10: Documentation — README logger-injection section

**Files:** Modify `README.md` (and any `doc.go`).

- [ ] **Step 1:** Add a "Logging" section documenting: default is silent until `SetVerbose`; `WithLogger(*slog.Logger)` + `SetLogger`; the `component=zftp` + `category` attributes; the level mapping (trace→Debug, Warning→Warn, Error→Error). Include the three bridge snippets from the spec, each marked "verify the bridge constructor against its current release."
- [ ] **Step 2:** `CGO_ENABLED=0 go build ./...` (docs-only, sanity). Commit `docs: document per-session slog logger injection`.

---

### Task 11: Final gate, PR, project item

- [ ] **Step 1: Full gate (show output)**

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./... && staticcheck ./... && go test -race ./... && govulncheck ./...
```
Expected: all green; `govulncheck` → "No vulnerabilities found."

- [ ] **Step 2: Adversarial review** — `cavecrew-reviewer` over `git diff main...feat/logger-injection` (dispatch with `isolation: worktree` so it does not `git checkout` the shared tree — see workflow-prefs gotcha).

- [ ] **Step 3: Project + PR** — create a Project #3 draft item ("Per-session slog logger injection"), promote draft→issue in place (`convertProjectV2DraftIssueItemToIssue`, `itemId`+`repositoryId` only), set a Priority + Status=In Progress; push; `gh pr create --base main` with `Closes #N`; wait for CI green; merge (`--merge --delete-branch`); confirm board → Done; update project memory.

---

## Self-Review

**Spec coverage:** slog bridge ✓(T2,T3,T10) · per-session ✓(T3) · bitmask prefilter + category attr ✓(T2) · `*slog.Logger`+`slog.Default()` lazy ✓(T2,T3) · drop Fatal/Panic ✓(T7; Fatal only in stdlib-`log` example_test, untouched) · component on every record ✓(T2) · go 1.21 + CI ✓(T1) · call-site threading incl. utils/transfer ✓(T4–T8) · tests (prefilter, mapping, attrs, always-on, default, source, race, isolation) ✓(T2,T3) · breaking-change/migration ✓(T1,T9) · README bridges ✓(T10) · gate ✓(T11).

**Placeholder scan:** the `wantLine` placeholder in the source test is explicitly annotated to be deleted; all other steps carry real code/commands.

**Type consistency:** `New(*slog.Logger, Level) *Logger`; `SetSlog`/`SetLevel`/`Level()`; emitters `Xf(string, ...any)`/`X(...any)`; `WithLogger(*slog.Logger) Option`; `(*FTPSession).SetLogger(*slog.Logger)`; `check(*bufio.Reader, *log.Logger)`; `childConnection.lg`. Used consistently across tasks.
