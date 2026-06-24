// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	zftp "gopkg.in/ro-ag/zftp.v2"
	"gopkg.in/ro-ag/zftp.v2/internal/mockzos"
)

// TestList_TruncatedData_Errors guards against silent truncation: z/OS aborts a
// failed transfer with a TCP RST (it uses a clean FIN only on success), so a reset
// on the data connection means the listing is incomplete and must surface as an
// error — even when the control reply still says 250.
func TestList_TruncatedData_Errors(t *testing.T) {
	s, srv := dialMock(t)
	srv.DataFor("LIST", "",
		"Volume Unit    Referred Ext Used Recfm Lrecl BlkSz Dsorg Dsname\r\n"+
			"FA00FF 3390   2023/06/02  1  360  VB    8004 27998  PS  'ABCD.EF.SEQ'\r\n")
	srv.TruncateData("LIST") // data connection aborted with RST; control still says 250

	if _, err := s.List("ABCD.EF.*"); err == nil {
		t.Fatal("List over a reset (truncated) data connection must error, got nil")
	}
	// The terminal control reply was left unconsumed, so the control stream is
	// desynchronized; the session must be closed rather than reused out of phase.
	if !s.IsClosed() {
		t.Fatal("session must be closed after a data-stream failure to avoid control-stream desync")
	}
}

// errWriter fails every write, to deterministically force a data-stream error
// mid-retrieve without depending on TCP reset timing.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("sink failure") }

// TestRetrieve_SinkError_ClosesSession is the RETR/STOR twin of the LIST case: a
// data-stream failure (here, the destination writer erroring) leaves the
// transfer's terminal reply unconsumed, so the session must be closed rather than
// left desynchronized.
func TestRetrieve_SinkError_ClosesSession(t *testing.T) {
	s, srv := dialMock(t)
	srv.DataFor("RETR", "BIG.BIN", strings.Repeat("z", 64*1024))

	if _, err := s.RetrieveIO("BIG.BIN", errWriter{}, zftp.TypeBinary); err == nil {
		t.Fatal("RetrieveIO with a failing sink must error, got nil")
	}
	if !s.IsClosed() {
		t.Fatal("session must be closed after a failed transfer to avoid control-stream desync")
	}
}

// TestList_ConcurrentClose_Aborts verifies that a LIST whose data scan is in
// flight when the session is closed from another goroutine (e.g. the SIGINT
// handler) reports an abort error rather than silently returning a partial
// listing as success.
func TestList_ConcurrentClose_Aborts(t *testing.T) {
	s, srv := dialMock(t)
	srv.DataFor("LIST", "", "FA00FF 3390   2023/06/02  1  360  VB    8004 27998  PS  'ABCD.EF.SEQ'\r\n")
	srv.HangData("LIST") // server holds the data conn open after the line, so the scan blocks

	listErr := make(chan error, 1)
	go func() {
		_, err := s.List("ABCD.EF.*")
		listErr <- err
	}()
	time.Sleep(250 * time.Millisecond) // let the scan drain the line and block on the next read

	closed := make(chan struct{})
	go func() { _ = s.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("Close hung during a concurrent List")
	}

	select {
	case err := <-listErr:
		if err == nil {
			t.Error("List aborted by a concurrent Close must error, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("List did not return after Close")
	}
}

// TestList_ReplyTimeout_ClosesSession verifies the post-transfer reply read is
// bounded: when the data is delivered and closed cleanly but the closing 250
// never arrives (and the control link stays open), the call must time out, error,
// and close the session rather than hang forever.
func TestList_ReplyTimeout_ClosesSession(t *testing.T) {
	srv := mockzos.New(t)
	s, err := zftp.Open(srv.Addr(), zftp.WithReplyTimeout(250*time.Millisecond))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Login("ME", "PW"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	srv.DataFor("LIST", "", "FA00FF 3390   2023/06/02  1  360  VB    8004 27998  PS  'ABCD.EF.SEQ'\r\n")
	srv.WithholdReplyAfterData("LIST") // data delivered + closed, but no 250; control stays open

	runWithTimeout(t, 5*time.Second, func() {
		if _, err := s.List("ABCD.EF.*"); err == nil {
			t.Error("List must error when the closing reply never arrives, got nil")
		}
	})
	if !s.IsClosed() {
		t.Error("session must be closed after the post-transfer reply read times out")
	}
}

// TestSession_ConcurrentRetrieveAndClose_NoPanic exercises the data-connection
// Read path (used by io.Copy for RETR) racing a concurrent Close. It must be
// race-free and not panic — the guard for dropping childConnection's mutex in
// favor of net.Conn's own concurrency safety plus the close interrupt.
func TestSession_ConcurrentRetrieveAndClose_NoPanic(t *testing.T) {
	s, srv := dialMock(t)
	srv.DataFor("RETR", "BIG.BIN", strings.Repeat("0123456789abcdef", 16*1024)) // 256 KiB

	runWithTimeout(t, 10*time.Second, func() {
		var wg sync.WaitGroup
		wg.Go(func() {
			var buf bytes.Buffer
			_, _ = s.RetrieveIO("BIG.BIN", &buf, zftp.TypeBinary)
		})
		wg.Go(func() {
			_ = s.Close()
		})
		wg.Wait()
	})
}
