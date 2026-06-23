// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"bytes"
	"strings"
	"testing"

	zftp "gopkg.in/ro-ag/zftp.v2"
)

// TestRetrieveIO_Accepts226Completion verifies a retrieve succeeds when the
// server closes the transfer with 226 (CodeClosingDataConn) instead of 250. Real
// z/OS and RFC 959 servers may send either, so the post-transfer reply read must
// accept both.
func TestRetrieveIO_Accepts226Completion(t *testing.T) {
	s, srv := dialMock(t)
	srv.DataFor("RETR", "MY.BIN", "payload-bytes")
	srv.CompletionReply("RETR", "226 closing data connection; transfer complete")

	var buf bytes.Buffer
	if _, _, err := s.RetrieveIO("MY.BIN", &buf, zftp.TypeBinary); err != nil {
		t.Fatalf("RetrieveIO with a 226 completion: %v", err)
	}
	if buf.String() != "payload-bytes" {
		t.Errorf("data = %q, want payload-bytes", buf.String())
	}
}

// TestStoreIO_Accepts226Completion verifies a store succeeds when the server
// closes the transfer with 226 instead of 250.
func TestStoreIO_Accepts226Completion(t *testing.T) {
	s, srv := dialMock(t)
	srv.CompletionReply("STOR", "226 closing data connection; transfer complete")

	if _, _, err := s.StoreIO("OUT.BIN", strings.NewReader("data"), zftp.TypeBinary); err != nil {
		t.Fatalf("StoreIO with a 226 completion: %v", err)
	}
	if stored, ok := srv.Stored("OUT.BIN"); !ok || string(stored) != "data" {
		t.Errorf("stored = %q (ok=%v), want data", stored, ok)
	}
}

// TestList_Accepts226Completion verifies a LIST succeeds when the data transfer
// is closed with 226 instead of 250 — the same completion path (confirmData) as
// RETR/STOR.
func TestList_Accepts226Completion(t *testing.T) {
	s, srv := dialMock(t)
	listing := "Volume Unit    Referred Ext Used Recfm Lrecl BlkSz Dsorg Dsname\r\n" +
		"FA00FF 3390   2023/06/02  1  360  VB    8004 27998  PS  'ABCD.EF.SEQ'\r\n"
	srv.DataFor("LIST", "", listing)
	srv.CompletionReply("LIST", "226 closing data connection")

	lines, err := s.List("ABCD.EF.*")
	if err != nil {
		t.Fatalf("List with a 226 completion: %v", err)
	}
	if len(lines) != 2 {
		t.Errorf("got %d lines, want 2 (header + 1 row)", len(lines))
	}
}
