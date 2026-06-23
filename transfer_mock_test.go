// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	zftp "gopkg.in/ro-ag/zftp.v2"
)

// TestRetrieveIO_Binary checks a binary retrieve is byte-for-byte exact,
// including NUL and high bytes that ASCII handling would corrupt.
func TestRetrieveIO_Binary(t *testing.T) {
	s, srv := dialMock(t)
	want := []byte{0x00, 0x01, 0x02, 0xFF, 'h', 'i', 0x0A, 0x00, 0xC1}
	srv.DataFor("RETR", "MY.BIN", string(want))

	var buf bytes.Buffer
	n, _, err := s.RetrieveIO("MY.BIN", &buf, zftp.TypeBinary)
	if err != nil {
		t.Fatalf("RetrieveIO: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("data = % x, want % x", buf.Bytes(), want)
	}
	if n != int64(len(want)) {
		t.Errorf("n = %d, want %d", n, len(want))
	}
}

// TestStoreIO_ASCII_CRLF checks ASCII store converts each LF-terminated source
// line to CRLF on the wire, and that the reported size matches the bytes sent.
func TestStoreIO_ASCII_CRLF(t *testing.T) {
	s, srv := dialMock(t)
	src := strings.NewReader("alpha\nbeta\ngamma") // LF input, no trailing newline

	n, _, err := s.StoreIO("OUT.TXT", src, zftp.TypeAscii)
	if err != nil {
		t.Fatalf("StoreIO: %v", err)
	}

	got, ok := srv.Stored("OUT.TXT")
	if !ok {
		t.Fatal("server captured no upload")
	}
	want := "alpha\r\nbeta\r\ngamma\r\n"
	if string(got) != want {
		t.Errorf("stored = %q, want %q", got, want)
	}
	if n != int64(len(want)) {
		t.Errorf("n = %d, want %d", n, len(want))
	}
}

// TestStoreIO_RestoresType verifies the transfer type is restored after an ASCII
// store: the login default is binary (TYPE I), so a TYPE I must follow the
// store's TYPE A.
func TestStoreIO_RestoresType(t *testing.T) {
	s, srv := dialMock(t)
	if _, _, err := s.StoreIO("F", strings.NewReader("x"), zftp.TypeAscii); err != nil {
		t.Fatalf("StoreIO: %v", err)
	}

	cmds := srv.Commands()
	a := cmdIndex(cmds, "TYPE A")
	if a < 0 {
		t.Fatal("expected a TYPE A for the ASCII store")
	}
	restored := false
	for _, c := range cmds[a+1:] {
		if strings.EqualFold(strings.TrimSpace(c), "TYPE I") {
			restored = true
			break
		}
	}
	if !restored {
		t.Errorf("transfer type not restored to TYPE I after store; commands=%v", cmds)
	}
}

// TestRetrieveIOAt_Offset verifies REST is sent before the transfer command when
// an offset is requested.
func TestRetrieveIOAt_Offset(t *testing.T) {
	s, srv := dialMock(t)
	srv.DataFor("RETR", "BIG.SEQ", "tail-bytes")

	var buf bytes.Buffer
	if _, _, err := s.RetrieveIOAt("BIG.SEQ", &buf, zftp.TypeBinary, 4096); err != nil {
		t.Fatalf("RetrieveIOAt: %v", err)
	}
	if !hasCmd(srv.Commands(), "REST 4096") {
		t.Errorf("expected REST 4096 in command sequence; got %v", srv.Commands())
	}
	if buf.String() != "tail-bytes" {
		t.Errorf("data = %q, want tail-bytes", buf.String())
	}
}

// TestRetrieveIOAt_AsciiOffsetRejected verifies that requesting a byte-offset
// resume in ASCII mode is rejected with ErrAsciiResumeUnsupported before any I/O:
// in TYPE A the server translates EOL/codepage, so a byte offset would slice
// mid-record and corrupt the data. No control command (REST/RETR/TYPE) may be
// sent and nothing may be written to the destination.
func TestRetrieveIOAt_AsciiOffsetRejected(t *testing.T) {
	s, srv := dialMock(t)
	srv.DataFor("RETR", "BIG.SEQ", "should-not-be-delivered")

	before := len(srv.Commands())
	var buf bytes.Buffer
	_, _, err := s.RetrieveIOAt("BIG.SEQ", &buf, zftp.TypeAscii, 100)
	if !errors.Is(err, zftp.ErrAsciiResumeUnsupported) {
		t.Fatalf("err = %v, want ErrAsciiResumeUnsupported", err)
	}
	if after := srv.Commands(); len(after) != before {
		t.Errorf("rejected ASCII resume sent control command(s): %v", after[before:])
	}
	if buf.Len() != 0 {
		t.Errorf("wrote %d byte(s) on rejected ASCII resume, want 0", buf.Len())
	}
}

// TestStoreIOAt_AsciiOffsetRejected verifies that an ASCII store with a byte
// offset is rejected with ErrAsciiResumeUnsupported before any I/O: no control
// command (REST/STOR/TYPE) may be sent and nothing may be stored on the server.
func TestStoreIOAt_AsciiOffsetRejected(t *testing.T) {
	s, srv := dialMock(t)

	before := len(srv.Commands())
	_, _, err := s.StoreIOAt("OUT.TXT", strings.NewReader("alpha\nbeta\n"), zftp.TypeAscii, 100)
	if !errors.Is(err, zftp.ErrAsciiResumeUnsupported) {
		t.Fatalf("err = %v, want ErrAsciiResumeUnsupported", err)
	}
	if after := srv.Commands(); len(after) != before {
		t.Errorf("rejected ASCII resume sent control command(s): %v", after[before:])
	}
	if _, ok := srv.Stored("OUT.TXT"); ok {
		t.Errorf("server stored data on a rejected ASCII resume")
	}
}

// TestStoreIOAt_BinaryOffset is a regression guard: image/binary resume must keep
// working — REST <n> is sent before STOR and the exact bytes are stored.
func TestStoreIOAt_BinaryOffset(t *testing.T) {
	s, srv := dialMock(t)
	payload := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x10}

	n, _, err := s.StoreIOAt("BIG.BIN", bytes.NewReader(payload), zftp.TypeBinary, 2048)
	if err != nil {
		t.Fatalf("StoreIOAt: %v", err)
	}
	if !hasCmd(srv.Commands(), "REST 2048") {
		t.Errorf("expected REST 2048 in command sequence; got %v", srv.Commands())
	}
	stored, ok := srv.Stored("BIG.BIN")
	if !ok || !bytes.Equal(stored, payload) {
		t.Fatalf("stored = % x (ok=%v), want % x", stored, ok, payload)
	}
	if n != int64(len(payload)) {
		t.Errorf("n = %d, want %d", n, len(payload))
	}
}
