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
