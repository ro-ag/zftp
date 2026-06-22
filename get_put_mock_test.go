// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"bytes"
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
