// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"bytes"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	zftp "gopkg.in/ro-ag/zftp.v2"
	"gopkg.in/ro-ag/zftp.v2/internal/mockzos"
)

// M1: a session that has never had its transfer type set (e.g. used before
// Login) must not restore to the zero TransferType, which would send the invalid
// "TYPE \x00". The connect-time default is ASCII (RFC 959 §3.1.1.3).
func TestTransfer_FreshSessionNeverSendsNullType(t *testing.T) {
	srv := mockzos.New(t)
	s, err := zftp.Open(srv.Addr()) // deliberately NO Login: currType stays unset
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	srv.DataFor("RETR", "FOO", "hello")
	var buf bytes.Buffer
	if _, err := s.RetrieveIO("FOO", &buf, zftp.TypeImage); err != nil {
		t.Fatalf("RetrieveIO on fresh session: %v", err)
	}
	for _, c := range srv.Commands() {
		if strings.ContainsRune(c, 0) {
			t.Fatalf("a TYPE command carried a NUL byte (currType zero value): %q", c)
		}
	}
}

// M2: a control reply that never arrives must not hang the caller forever. The
// data command (RETR/STOR) and REST are issued via SendCommand; every SendCommand
// round-trip must be bounded by the reply timeout.
func TestTransfer_StalledCommandReplyTimesOut(t *testing.T) {
	srv := mockzos.New(t)
	s, err := zftp.Open(srv.Addr(), zftp.WithReplyTimeout(300*time.Millisecond))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Login("ME", "PW"); err != nil {
		t.Fatalf("Login: %v", err)
	}

	srv.Withhold("RETR") // server consumes RETR but never sends a reply

	done := make(chan error, 1)
	go func() {
		var buf bytes.Buffer
		_, e := s.RetrieveIO("FOO", &buf, zftp.TypeImage)
		done <- e
	}()
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("expected a timeout error from the withheld RETR reply, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RetrieveIO hung on a withheld RETR reply: control read has no deadline")
	}
}

// M5: checkLast (and the exported CheckLast) must never report success on a
// closed session. Returning ("", nil) would let confirmData report a transfer
// whose terminal reply was never read as complete.
func TestCheckLast_ClosedSessionReturnsError(t *testing.T) {
	s, _ := dialMock(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_, err := s.CheckLast(zftp.CodeFileActionOK)
	if err == nil {
		t.Fatal("CheckLast on a closed session returned nil (false success)")
	}
	if !errors.Is(err, net.ErrClosed) {
		t.Fatalf("want net.ErrClosed, got %v", err)
	}
}

// M7: a passive transfer whose terminal control reply is not a completion code
// (e.g. 426 "transfer aborted", which z/OS may follow with a second reply such as
// 226) must close the session. FTP has no in-band control-stream resync, so a
// trailing reply left buffered would desync the next command ("shifted messages").
func TestTransfer_AbortCompletionReplyClosesSession(t *testing.T) {
	s, srv := dialMock(t)
	srv.DataFor("RETR", "FOO", "hello")
	srv.CompletionReply("RETR", "426 connection closed; transfer aborted")

	var buf bytes.Buffer
	_, err := s.RetrieveIO("FOO", &buf, zftp.TypeImage)
	if err == nil {
		t.Fatal("expected an error from the 426 abort completion, got nil")
	}
	if !s.IsClosed() {
		t.Fatal("session left open after a non-completion (426) transfer reply: a trailing reply will desync the next command")
	}
}

// Minor: a failed login must not leave the attempted username recorded; User
// should reflect only a successful authentication.
func TestLogin_FailedLoginDoesNotRecordUser(t *testing.T) {
	srv := mockzos.New(t)
	s, err := zftp.Open(srv.Addr())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	srv.Script("PASS", "530 not logged in")
	if err := s.Login("ME", "BADPW"); err == nil {
		t.Fatal("expected a login failure")
	}
	if u := s.User(); u != "" {
		t.Fatalf("User() = %q after a failed login, want empty", u)
	}
}
