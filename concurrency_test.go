// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	zftp "gopkg.in/ro-ag/zftp.v2"
)

// runWithTimeout runs fn in a goroutine and fails the test if it does not return
// within d. It turns a deadlock or a desynchronized control stream (which would
// otherwise hang the whole package until the global -timeout) into a fast, clear
// failure.
func runWithTimeout(t *testing.T, d time.Duration, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("operation did not complete within %s (possible deadlock or stream desync)", d)
	}
}

// TestSession_ConcurrentCommandsAndClose_RaceFree shares one *FTPSession across
// many goroutines issuing control round-trips, passive negotiations and transfer
// type changes while another goroutine closes the session. It must run cleanly
// under `go test -race`: every command-issuing path and Close has to serialize on
// the session mutex, and mutable fields (the control reader and currType) must
// not be touched concurrently.
func TestSession_ConcurrentCommandsAndClose_RaceFree(t *testing.T) {
	s, _ := dialMock(t)

	runWithTimeout(t, 10*time.Second, func() {
		var wg sync.WaitGroup

		// Concurrent control round-trips: each reads the shared control reader.
		for range 8 {
			wg.Go(func() {
				_, _ = s.SendCommand(zftp.CodeCmdOK, "NOOP")
			})
		}

		// Concurrent passive-mode negotiations (PASV round-trip + parse).
		for range 4 {
			wg.Go(func() {
				_, _ = s.SetPassiveMode()
			})
		}

		// Concurrent transfer-type writes (exercise the currType field).
		for i := range 4 {
			typ := zftp.TypeAscii
			if i%2 == 1 {
				typ = zftp.TypeBinary
			}
			wg.Go(func() {
				_ = s.SetType(typ)
			})
		}

		// A concurrent Close must not race the in-flight commands.
		wg.Go(func() {
			_ = s.Close()
		})

		wg.Wait()
	})
}

// TestSendCommandWithContext_TimeoutClosesSession reproduces the timeout-desync
// bug: when the server withholds a reply and the context deadline fires, the call
// must abort its own I/O, report an error, and mark the session closed so the
// control stream cannot be read one reply ahead by a later command. A subsequent
// command must then fail fast rather than block or consume a stale reply.
func TestSendCommandWithContext_TimeoutClosesSession(t *testing.T) {
	s, srv := dialMock(t)
	srv.Withhold("STAT") // the server receives STAT but never replies

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	if _, err := s.SendCommandWithContext(ctx, zftp.CodeSysStatus, "STAT"); err == nil {
		t.Fatal("SendCommandWithContext: want a timeout error, got nil")
	}

	if !s.IsClosed() {
		t.Fatal("session must be closed after a control-stream timeout")
	}

	// The desync this guards against: a later command must not block on, or read,
	// the stale reply left behind by the timed-out command.
	runWithTimeout(t, 2*time.Second, func() {
		if _, err := s.SendCommand(zftp.CodeCmdOK, "NOOP"); err == nil {
			t.Error("SendCommand on a closed session: want an error, got nil")
		}
	})
}

// TestClose_UnblocksStalledCommand guards against a deadlock regression: a
// SendCommand with no deadline blocks reading the reply while holding the session
// mutex. If a peer goes silent, a concurrent Close must still be able to tear the
// session down (by interrupting the in-flight read) instead of starving forever
// on the mutex.
func TestClose_UnblocksStalledCommand(t *testing.T) {
	s, srv := dialMock(t)
	srv.Withhold("STAT") // STAT is received but never answered

	cmdDone := make(chan error, 1)
	go func() {
		_, err := s.SendCommand(zftp.CodeSysStatus, "STAT") // no deadline: blocks under s.mu
		cmdDone <- err
	}()
	time.Sleep(150 * time.Millisecond) // let the command acquire the mutex and block on the read

	// Close must not hang behind the stalled command.
	runWithTimeout(t, 3*time.Second, func() { _ = s.Close() })

	// And the stalled command must be released (with an error), not leaked.
	select {
	case err := <-cmdDone:
		if err == nil {
			t.Error("stalled SendCommand: want an error after Close, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("stalled SendCommand was not unblocked by Close")
	}
}

// TestSendCommand_PeerCloseClosesSession reproduces the EOF-misclassification
// bug: when the peer drops the control connection instead of replying, the read
// hits EOF. That is an unrecoverable I/O failure (not a valid FTP reply), so the
// session must be marked closed — not left open to read garbage on the next call.
func TestSendCommand_PeerCloseClosesSession(t *testing.T) {
	s, srv := dialMock(t)
	srv.Hangup("STAT") // server drops the control connection on STAT

	if _, err := s.SendCommand(zftp.CodeSysStatus, "STAT"); err == nil {
		t.Fatal("SendCommand: want an error when the peer closes the control connection, got nil")
	}
	if !s.IsClosed() {
		t.Fatal("session must be closed after a control-connection EOF")
	}
}

// TestList_ControlDropClosesSession covers the other control-stream reader: the
// post-transfer reply read in checkLast. If the peer drops the control connection
// after the data but before the closing reply, checkLast hits EOF and must close
// the session, since the stream is desynchronized for any later command.
func TestList_ControlDropClosesSession(t *testing.T) {
	s, srv := dialMock(t)
	srv.DataFor("LIST", "", "FA00FF 3390 PS 'ABCD.EF.SEQ'\r\n")
	srv.DropControlAfterData("LIST") // data is delivered, then the control conn is dropped (no 250)

	if _, err := s.List("ABCD.EF.*"); err == nil {
		t.Fatal("List: want an error when the closing reply never arrives, got nil")
	}
	if !s.IsClosed() {
		t.Fatal("session must be closed after the post-transfer reply read hits EOF")
	}
}

// TestSession_ConcurrentListAndClose_NoPanic runs a passive LIST whose data
// scan is in flight while the session is closed from another goroutine. Closing
// must interrupt the data scan promptly and must not panic or deadlock.
func TestSession_ConcurrentListAndClose_NoPanic(t *testing.T) {
	s, srv := dialMock(t)
	// A sizable listing so the data scan is still draining when Close lands.
	var listing strings.Builder
	for range 500 {
		listing.WriteString("FA00FF 3390   2023/06/02  1  360  VB    8004 27998  PS  'ABCD.EF.SEQ'\r\n")
	}
	srv.DataFor("LIST", "", listing.String())

	runWithTimeout(t, 10*time.Second, func() {
		var wg sync.WaitGroup
		wg.Go(func() {
			_, _ = s.List("ABCD.EF.*")
		})
		wg.Go(func() {
			_ = s.Close()
		})
		wg.Wait()
	})
}
