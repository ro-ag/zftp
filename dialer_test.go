// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
)

// pipeDialer hands the client end of an in-memory net.Pipe to Open, so the
// session can be driven entirely in-process with no real network.
type pipeDialer struct{ conn net.Conn }

func (p pipeDialer) DialContext(_ context.Context, _, _ string) (net.Conn, error) {
	return p.conn, nil
}

func TestOpen_WithInjectedDialer(t *testing.T) {
	client, server := net.Pipe()

	go func() {
		// Greeting the client reads during Open.
		_, _ = fmt.Fprint(server, "220 service ready for new user\r\n")
		// Drain whatever the client writes (e.g. on Close) until the pipe closes.
		_, _ = io.Copy(io.Discard, server)
	}()

	s, err := Open("fake-host:21", WithDialer(pipeDialer{client}))
	if err != nil {
		t.Fatalf("Open with injected dialer: %v", err)
	}
	if s.Conn() == nil {
		t.Fatal("session has no connection")
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestOpen_GreetingError(t *testing.T) {
	client, server := net.Pipe()

	go func() {
		// A non-220 greeting must surface as an error from Open.
		_, _ = fmt.Fprint(server, "421 service not available\r\n")
		_, _ = io.Copy(io.Discard, server)
	}()

	_, err := Open("fake-host:21", WithDialer(pipeDialer{client}))
	if err == nil {
		t.Fatal("expected error for non-220 greeting")
	}
	var re *ReturnError
	if !errors.As(err, &re) || re.ReturnCode() != 421 {
		t.Fatalf("want ReturnError rc=421, got %v", err)
	}
}
