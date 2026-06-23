// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
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

// TestGetAndGzip_RoundTrip retrieves a payload and compresses it on the fly, then
// asserts the resulting .gz decompresses byte-for-byte to the original — covering
// the gzip writer/footer path and VerifyGzSize, which previously had no test.
func TestGetAndGzip_RoundTrip(t *testing.T) {
	s, srv := dialMock(t)
	dir := t.TempDir()
	payload := []byte("hello z/OS - gzip round trip\nline two\nline three\n")
	srv.DataFor("RETR", "MY.DATA", string(payload))

	local := filepath.Join(dir, "out") // GetAndGzip appends ".gz"
	if err := s.GetAndGzip("MY.DATA", local, zftp.TypeBinary); err != nil {
		t.Fatalf("GetAndGzip: %v", err)
	}

	f, err := os.Open(local + ".gz")
	if err != nil {
		t.Fatalf("open gz: %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	got, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("read gz: %v", err)
	}
	if err := zr.Close(); err != nil {
		t.Errorf("gzip reader Close (checksum/size mismatch?): %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("decompressed = %q, want %q", got, payload)
	}
}

// TestGetAt_BinaryOffset_WritesAtOffset covers the binary resume path of the
// local-file GetAt: with a positive offset it must send REST <n>, seek the local
// file to the offset (not truncate), and write the retrieved bytes there, leaving
// the bytes already present before the offset intact.
func TestGetAt_BinaryOffset_WritesAtOffset(t *testing.T) {
	s, srv := dialMock(t)
	dir := t.TempDir()
	local := filepath.Join(dir, "out.bin")
	if err := os.WriteFile(local, []byte("HEAD"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv.DataFor("RETR", "BIG.BIN", "TAILDATA")

	if err := s.GetAt("BIG.BIN", local, zftp.TypeBinary, 4); err != nil {
		t.Fatalf("GetAt: %v", err)
	}
	if !hasCmd(srv.Commands(), "REST 4") {
		t.Errorf("expected REST 4 in command sequence; got %v", srv.Commands())
	}
	got, err := os.ReadFile(local)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "HEADTAILDATA" {
		t.Errorf("local file = %q, want HEADTAILDATA", got)
	}
}

// TestPutAt_BinaryOffset_UploadsFromOffset covers the binary resume path of the
// local-file PutAt: with a positive offset it must seek the source to the offset
// and send only the bytes from there, prefixed by REST <n>.
func TestPutAt_BinaryOffset_UploadsFromOffset(t *testing.T) {
	s, srv := dialMock(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	if err := os.WriteFile(src, []byte("HEADTAILDATA"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := s.PutAt(src, "BIG.BIN", zftp.TypeBinary, 4); err != nil {
		t.Fatalf("PutAt: %v", err)
	}
	if !hasCmd(srv.Commands(), "REST 4") {
		t.Errorf("expected REST 4 in command sequence; got %v", srv.Commands())
	}
	stored, ok := srv.Stored("BIG.BIN")
	if !ok || string(stored) != "TAILDATA" {
		t.Errorf("stored = %q (ok=%v), want TAILDATA", stored, ok)
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
