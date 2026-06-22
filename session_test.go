// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"testing"

	zftp "gopkg.in/ro-ag/zftp.v2"
	"gopkg.in/ro-ag/zftp.v2/internal/mockzos"
)

// dialMock starts an in-process mock z/OS FTP server, opens a real client
// session against it over loopback, logs in, and returns both so tests can
// script further responses and assert on captured state. Cleanup is registered
// with the test.
func dialMock(t *testing.T) (*zftp.FTPSession, *mockzos.Server) {
	t.Helper()
	srv := mockzos.New(t)
	s, err := zftp.Open(srv.Addr())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Login("ME", "PW"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	return s, srv
}

// TestSession_ListDatasets_EndToEnd exercises the full stack against the mock:
// dial, login (USER/PASS/PASV/TYPE/SITE/SYST), then a passive LIST whose data is
// parsed into dataset records — no mainframe involved.
func TestSession_ListDatasets_EndToEnd(t *testing.T) {
	s, srv := dialMock(t)

	listing := "Volume Unit    Referred Ext Used Recfm Lrecl BlkSz Dsorg Dsname\r\n" +
		"FA00FF 3390   2023/06/02  1  360  VB    8004 27998  PS  'ABCD.EF.SEQ'\r\n" +
		"Migrated                                                'ABCD.EF.MIGR'\r\n"
	srv.DataFor("LIST", "", listing)

	ds, err := s.ListDatasets("ABCD.EF.*")
	if err != nil {
		t.Fatalf("ListDatasets: %v", err)
	}
	if len(ds) != 2 {
		t.Fatalf("got %d datasets, want 2", len(ds))
	}
	if got := ds[0].Name(); got != "ABCD.EF.SEQ" {
		t.Errorf("ds[0].Name() = %q, want ABCD.EF.SEQ", got)
	}
	if !ds[0].IsSequential() {
		t.Errorf("ds[0] should be sequential (Dsorg=%q)", ds[0].Dsorg.String())
	}
	if !ds[1].IsMigrated() {
		t.Errorf("ds[1] should be migrated")
	}
}
