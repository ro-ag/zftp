// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"testing"

	"gopkg.in/ro-ag/zftp.v2/internal/log"
)

// An argument-less command must not carry a trailing space (RFC 959 command
// lines are "verb CRLF" with no arguments, e.g. "NOOP\r\n" not "NOOP \r\n").
func TestParseCommand_NoTrailingSpaceWhenArgless(t *testing.T) {
	lg := log.New(nil, log.None)
	if got := string(parseCommand(lg, "NOOP")); got != "NOOP\r\n" {
		t.Fatalf("argless command = %q, want %q", got, "NOOP\r\n")
	}
}

func TestParseCommand_WithArgsUnchanged(t *testing.T) {
	lg := log.New(nil, log.None)
	if got := string(parseCommand(lg, "RETR", "FOO")); got != "RETR FOO\r\n" {
		t.Fatalf("command with args = %q, want %q", got, "RETR FOO\r\n")
	}
}

// Name must not label an unknown/zero TransferType as "BINARY".
func TestTransferType_NameUnknownForZeroValue(t *testing.T) {
	if got := TransferType(0).Name(); got != "UNKNOWN" {
		t.Fatalf("TransferType(0).Name() = %q, want UNKNOWN", got)
	}
	if got := TypeAscii.Name(); got != "ASCII" {
		t.Fatalf("TypeAscii.Name() = %q, want ASCII", got)
	}
	if got := TypeImage.Name(); got != "BINARY" {
		t.Fatalf("TypeImage.Name() = %q, want BINARY", got)
	}
}
