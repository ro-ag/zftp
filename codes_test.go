// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"errors"
	"strings"
	"testing"

	zftp "gopkg.in/ro-ag/zftp.v2"
)

func TestSendCommand_MultilineReply(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("STAT", "211-status line one", "211-status line two", "211 end of status")

	msg, err := s.SendCommand(zftp.CodeSysStatus, "STAT")
	if err != nil {
		t.Fatalf("SendCommand: %v", err)
	}
	for _, want := range []string{"status line one", "status line two", "end of status"} {
		if !strings.Contains(msg, want) {
			t.Errorf("multiline reply missing %q in:\n%s", want, msg)
		}
	}
}

func TestSendCommand_ReturnError(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("DELE BAD.DSN", "550 dataset not found")

	_, err := s.SendCommand(zftp.CodeFileActionOK, "DELE", "BAD.DSN")
	if err == nil {
		t.Fatal("want error for unexpected return code")
	}

	var re *zftp.ReturnError
	if !errors.As(err, &re) {
		t.Fatalf("want *ReturnError, got %T", err)
	}
	if re.ReturnCode() != zftp.CodeFileActionNotTakenPerm { // 550
		t.Errorf("ReturnCode() = %d, want 550", re.ReturnCode())
	}
	if !strings.Contains(re.Error(), "550") || !strings.Contains(re.Error(), "250") {
		t.Errorf("Error() should mention got/want codes: %q", re.Error())
	}
}

func TestSendCommand_Success(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("NOOP", "200 looks good")
	msg, err := s.SendCommand(zftp.CodeCmdOK, "NOOP")
	if err != nil {
		t.Fatalf("SendCommand: %v", err)
	}
	if !strings.Contains(msg, "looks good") {
		t.Errorf("msg = %q", msg)
	}
}

// TestSendCommand_DeceptiveMidlineNotTerminator guards the RFC 959 §4.2
// terminator rule: a multiline reply ends only on a line that repeats the
// OPENING reply code followed by a space. A continuation line that merely looks
// like a terminator — three digits and a space, but a DIFFERENT code (here a
// z/OS message quoting "550 ...") — must be treated as a continuation, not the
// end. Otherwise the reply is truncated and the real terminator line desyncs the
// control stream for the next command.
func TestSendCommand_DeceptiveMidlineNotTerminator(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("STAT", "211-begin", "550 not the end", "211 actual end")
	srv.Script("NOOP", "200 its own reply")

	msg, err := s.SendCommand(zftp.CodeSysStatus, "STAT") // expects 211
	if err != nil {
		t.Fatalf("STAT: unexpected error (reply mis-terminated?): %v", err)
	}
	for _, want := range []string{"begin", "550 not the end", "actual end"} {
		if !strings.Contains(msg, want) {
			t.Errorf("reply truncated: missing %q in:\n%s", want, msg)
		}
	}

	// Core desync guard: the leftover real terminator must not bleed into the
	// next command — NOOP must read its OWN reply.
	msg2, err := s.SendCommand(zftp.CodeCmdOK, "NOOP") // expects 200
	if err != nil {
		t.Fatalf("NOOP after multiline: control stream desynced: %v", err)
	}
	if !strings.Contains(msg2, "its own reply") {
		t.Errorf("NOOP read a stale reply (desync): %q", msg2)
	}
}

// TestSendCommand_ShortContinuationLineRetained verifies that a continuation
// line shorter than 4 bytes (here a blank line) inside a multiline block is
// retained in the response — not silently dropped — and that parsing still
// terminates on the real terminator and leaves the stream in sync.
func TestSendCommand_ShortContinuationLineRetained(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("STAT", "211-header", "", "211 footer")
	srv.Script("NOOP", "200 still in sync")

	msg, err := s.SendCommand(zftp.CodeSysStatus, "STAT") // expects 211
	if err != nil {
		t.Fatalf("STAT: unexpected error: %v", err)
	}
	for _, want := range []string{"header", "footer"} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in:\n%s", want, msg)
		}
	}
	// The blank continuation line is retained as an empty line between the two.
	if !strings.Contains(msg, "header\n\nfooter") {
		t.Errorf("blank continuation line dropped; got:\n%q", msg)
	}

	// The reply terminated correctly, so the next command stays in sync.
	if _, err := s.SendCommand(zftp.CodeCmdOK, "NOOP"); err != nil {
		t.Fatalf("NOOP after short continuation line: desync: %v", err)
	}
}
