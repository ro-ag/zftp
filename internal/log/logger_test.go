// SPDX-License-Identifier: Apache-2.0

package log

import (
	"context"
	"log/slog"
	"math/bits"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// flags lists the four independent logging categories. None and All are derived
// from these, so the tests assert their relationship rather than listing them.
var flags = []Level{ServerLevel, PassiveLevel, CommandLevel, DebugLevel}

// capCore is the shared record store behind a capture handler and any
// WithAttrs-derived children.
type capCore struct {
	mu      sync.Mutex
	records []slog.Record
}

// capture is a slog.Handler that records every record it handles, honoring
// attributes accumulated through WithAttrs (e.g. the component tag added by
// Logger via slog.Logger.With). A faithful handler is required to observe those.
type capture struct {
	core  *capCore
	attrs []slog.Attr
}

func newCapture() *capture { return &capture{core: &capCore{}} }

func (c *capture) Enabled(context.Context, slog.Level) bool { return true }

func (c *capture) Handle(_ context.Context, r slog.Record) error {
	rec := r.Clone()
	rec.AddAttrs(c.attrs...)
	c.core.mu.Lock()
	defer c.core.mu.Unlock()
	c.core.records = append(c.core.records, rec)
	return nil
}

func (c *capture) WithAttrs(as []slog.Attr) slog.Handler {
	merged := append(append([]slog.Attr(nil), c.attrs...), as...)
	return &capture{core: c.core, attrs: merged}
}

func (c *capture) WithGroup(string) slog.Handler { return c }

func (c *capture) snapshot() []slog.Record {
	c.core.mu.Lock()
	defer c.core.mu.Unlock()
	return append([]slog.Record(nil), c.core.records...)
}

// attrMap flattens a record's attributes into a map for assertions.
func attrMap(r slog.Record) map[string]string {
	m := map[string]string{}
	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value.String()
		return true
	})
	return m
}

// --- ZH06 invariants: the four categories are distinct single bits ---

func TestLevelsAreDistinctSingleBits(t *testing.T) {
	for _, l := range flags {
		if got := bits.OnesCount32(uint32(l)); got != 1 {
			t.Errorf("level %d must be a single bit, has %d bits set", l, got)
		}
	}

	union := ServerLevel | PassiveLevel | CommandLevel | DebugLevel
	if got := bits.OnesCount32(uint32(union)); got != 4 {
		t.Errorf("union of the four levels must occupy 4 distinct bits, has %d (union=%d)", got, union)
	}

	for i := range flags {
		for j := i + 1; j < len(flags); j++ {
			if flags[i]&flags[j] != 0 {
				t.Errorf("levels %d and %d overlap (AND=%d)", flags[i], flags[j], flags[i]&flags[j])
			}
		}
	}
}

func TestNoneIsZero(t *testing.T) {
	if None != 0 {
		t.Errorf("None must be 0, got %d", None)
	}
}

func TestAllIsUnionOfTheFourLevels(t *testing.T) {
	want := ServerLevel | PassiveLevel | CommandLevel | DebugLevel
	if All != want {
		t.Errorf("All must equal the OR of the four levels (%d), got %d", want, All)
	}
}

// --- ZH06 behavioral invariants, expressed against the Logger bitmask ---

func TestEnabledCommandDoesNotEnableOthers(t *testing.T) {
	lg := New(nil, CommandLevel)
	if !lg.enabled(CommandLevel) {
		t.Error("CommandLevel must be enabled")
	}
	for _, l := range []Level{ServerLevel, PassiveLevel, DebugLevel} {
		if lg.enabled(l) {
			t.Errorf("level %d must NOT be enabled by CommandLevel", l)
		}
	}
}

func TestEnabledDebugDoesNotEnableOthers(t *testing.T) {
	lg := New(nil, DebugLevel)
	if !lg.enabled(DebugLevel) {
		t.Error("DebugLevel must be enabled")
	}
	for _, l := range []Level{ServerLevel, PassiveLevel, CommandLevel} {
		if lg.enabled(l) {
			t.Errorf("level %d must NOT be enabled by DebugLevel", l)
		}
	}
}

func TestEnabledAllEnablesEveryLevel(t *testing.T) {
	lg := New(nil, All)
	for _, l := range flags {
		if !lg.enabled(l) {
			t.Errorf("level %d must be enabled by All", l)
		}
	}
}

func TestEnabledNoneDisablesEveryLevel(t *testing.T) {
	lg := New(nil, None)
	for _, l := range flags {
		if lg.enabled(l) {
			t.Errorf("level %d must be disabled by None", l)
		}
	}
}

// --- slog emission behavior ---

func TestLogger_PrefilterEmitsOnlyEnabledCategory(t *testing.T) {
	cap := newCapture()
	lg := New(slog.New(cap), CommandLevel)

	lg.Commandf("PASS %s", "secret")
	lg.Serverf("220 ready") // ServerLevel not enabled
	lg.Passivef("227 ...")  // PassiveLevel not enabled
	lg.Debugf("trace")      // DebugLevel not enabled

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
	cap := newCapture()
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
	cap := newCapture()
	lg := New(slog.New(cap), DebugLevel)

	lg.Debugf("x")

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
	cap := newCapture()
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
	lg := New(slog.New(newCapture()), All)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for j := range 200 {
				lg.Commandf("c%d", j)
				lg.SetLevel(All)
				lg.SetSlog(slog.New(newCapture()))
			}
		})
	}
	wg.Wait()
}
