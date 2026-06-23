// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	zftp "gopkg.in/ro-ag/zftp.v2"
)

// TestPutGet_BinaryRoundTrip exercises the file-based Put and Get against the
// mock: Put uploads a local file (captured by the server), Get downloads it back
// to a fresh local file. Binary content must survive byte-for-byte.
func TestPutGet_BinaryRoundTrip(t *testing.T) {
	s, srv := dialMock(t)
	dir := t.TempDir()

	content := []byte{0x00, 'M', 'V', 'S', 0xFF, 0x10, '\n', 0x00}
	src := filepath.Join(dir, "src.bin")
	if err := os.WriteFile(src, content, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := s.Put(src, "USER.UPLOAD.BIN", zftp.TypeBinary); err != nil {
		t.Fatalf("Put: %v", err)
	}
	stored, ok := srv.Stored("USER.UPLOAD.BIN")
	if !ok || !bytes.Equal(stored, content) {
		t.Fatalf("uploaded = % x (ok=%v), want % x", stored, ok, content)
	}

	srv.DataFor("RETR", "USER.UPLOAD.BIN", string(content))
	dst := filepath.Join(dir, "out.bin")
	if err := s.Get("USER.UPLOAD.BIN", dst, zftp.TypeBinary); err != nil {
		t.Fatalf("Get: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("downloaded = % x, want % x", got, content)
	}
}

// TestGetAt_AsciiOffsetRejected_LocalUntouched verifies GetAt rejects an ASCII
// byte-offset resume before touching the local file: the destination (which does
// not exist yet) must not be created, and no control command may be sent.
func TestGetAt_AsciiOffsetRejected_LocalUntouched(t *testing.T) {
	s, srv := dialMock(t)
	srv.DataFor("RETR", "BIG.SEQ", "tail-bytes")
	dir := t.TempDir()
	local := filepath.Join(dir, "out.txt") // intentionally absent

	before := len(srv.Commands())
	err := s.GetAt("BIG.SEQ", local, zftp.TypeAscii, 100)
	if !errors.Is(err, zftp.ErrAsciiResumeUnsupported) {
		t.Fatalf("err = %v, want ErrAsciiResumeUnsupported", err)
	}
	if _, statErr := os.Stat(local); !os.IsNotExist(statErr) {
		t.Errorf("local file was created/touched on rejected ASCII resume (stat err = %v)", statErr)
	}
	if after := srv.Commands(); len(after) != before {
		t.Errorf("rejected ASCII resume sent control command(s): %v", after[before:])
	}
}

// TestPutAt_AsciiOffsetRejected_BeforeOpen verifies PutAt rejects an ASCII
// byte-offset resume before opening/seeking the source: pointing it at a
// non-existent source must still return ErrAsciiResumeUnsupported (not an
// open/seek error), proving the guard precedes local-file and network I/O.
func TestPutAt_AsciiOffsetRejected_BeforeOpen(t *testing.T) {
	s, srv := dialMock(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "does-not-exist.txt") // intentionally absent

	before := len(srv.Commands())
	err := s.PutAt(src, "OUT.TXT", zftp.TypeAscii, 100)
	if !errors.Is(err, zftp.ErrAsciiResumeUnsupported) {
		t.Fatalf("err = %v, want ErrAsciiResumeUnsupported (guard must precede opening the source)", err)
	}
	if after := srv.Commands(); len(after) != before {
		t.Errorf("rejected ASCII resume sent control command(s): %v", after[before:])
	}
	if _, ok := srv.Stored("OUT.TXT"); ok {
		t.Errorf("server stored data on a rejected ASCII resume")
	}
}
