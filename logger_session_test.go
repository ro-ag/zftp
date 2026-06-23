// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	ilog "gopkg.in/ro-ag/zftp.v2/internal/log"
)

// capHandler counts the records it handles, bucketed by their category attribute.
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

func (c *capHandler) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.cats)
}

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

// WithLogger wires the option through the construction path (newSession).
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
