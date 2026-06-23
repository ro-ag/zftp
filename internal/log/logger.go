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
// membership with the category emitters. Each flag is a distinct bit, so enabling
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

func (l *Logger) Command(v ...any) {
	if l.enabled(CommandLevel) {
		l.emit(slog.LevelDebug, fmt.Sprint(v...), catCommand)
	}
}

func (l *Logger) Commandf(format string, v ...any) {
	if l.enabled(CommandLevel) {
		l.emit(slog.LevelDebug, fmt.Sprintf(format, v...), catCommand)
	}
}

func (l *Logger) Server(v ...any) {
	if l.enabled(ServerLevel) {
		l.emit(slog.LevelDebug, fmt.Sprint(v...), catServer)
	}
}

func (l *Logger) Serverf(format string, v ...any) {
	if l.enabled(ServerLevel) {
		l.emit(slog.LevelDebug, fmt.Sprintf(format, v...), catServer)
	}
}

func (l *Logger) Passive(v ...any) {
	if l.enabled(PassiveLevel) {
		l.emit(slog.LevelDebug, fmt.Sprint(v...), catPassive)
	}
}

func (l *Logger) Passivef(format string, v ...any) {
	if l.enabled(PassiveLevel) {
		l.emit(slog.LevelDebug, fmt.Sprintf(format, v...), catPassive)
	}
}

func (l *Logger) Debug(v ...any) {
	if l.enabled(DebugLevel) {
		l.emit(slog.LevelDebug, fmt.Sprint(v...), catDebug)
	}
}

func (l *Logger) Debugf(format string, v ...any) {
	if l.enabled(DebugLevel) {
		l.emit(slog.LevelDebug, fmt.Sprintf(format, v...), catDebug)
	}
}

func (l *Logger) Warning(v ...any)            { l.emit(slog.LevelWarn, fmt.Sprint(v...)) }
func (l *Logger) Warningf(f string, v ...any) { l.emit(slog.LevelWarn, fmt.Sprintf(f, v...)) }
func (l *Logger) Error(v ...any)              { l.emit(slog.LevelError, fmt.Sprint(v...)) }
func (l *Logger) Errorf(f string, v ...any)   { l.emit(slog.LevelError, fmt.Sprintf(f, v...)) }
