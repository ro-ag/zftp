// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"testing"

	zftp "gopkg.in/ro-ag/zftp.v2"
)

func TestNList_Trims(t *testing.T) {
	s, srv := dialMock(t)
	srv.DataFor("NLST", "", "  ABCD.EF.ONE  \r\nABCD.EF.TWO\r\n")

	names, err := s.NList("ABCD.EF.*")
	if err != nil {
		t.Fatalf("NList: %v", err)
	}
	want := []string{"ABCD.EF.ONE", "ABCD.EF.TWO"}
	if len(names) != len(want) {
		t.Fatalf("got %d names %v, want %d", len(names), names, len(want))
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestListPds(t *testing.T) {
	s, srv := dialMock(t)
	listing := " Name     VV.MM   Created       Changed      Size  Init   Mod   Id\r\n" +
		"ARTIST    01.00 2021/08/20 2021/08/20 07:51     6     6     0 A99993\r\n" +
		"CBL0001   01.08 2021/06/09 2021/06/14 15:17    74    73     0 JBISTI\r\n"
	srv.DataFor("LIST", "", listing)

	members, err := s.ListPds("MY.SOURCE.PDS")
	if err != nil {
		t.Fatalf("ListPds: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("got %d members, want 2", len(members))
	}
	if members[0].Name.String() != "ARTIST" || members[0].Id.String() != "A99993" {
		t.Errorf("members[0] = %s", members[0].String())
	}
	if members[1].Name.String() != "CBL0001" {
		t.Errorf("members[1].Name = %q, want CBL0001", members[1].Name.String())
	}
}

func TestListSpool(t *testing.T) {
	s, srv := dialMock(t)
	listing := "JOBNAME  JOBID    OWNER    STATUS CLASS\r\n" +
		"ANOTHER  JOB06184 Z33500   OUTPUT A        RC=0000 4 spool files\r\n" +
		"JCL2     JOB06318 Z33935   OUTPUT A        ABEND=000 7 spool files\r\n"
	srv.DataFor("LIST", "", listing)

	jobs, err := s.ListSpool("*")
	if err != nil {
		t.Fatalf("ListSpool: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2", len(jobs))
	}
	if jobs[0].Name.String() != "ANOTHER" || jobs[0].JobId.String() != "JOB06184" {
		t.Errorf("jobs[0] = %s", jobs[0].String())
	}
	if jobs[0].Owner.String() != "Z33500" {
		t.Errorf("jobs[0].Owner = %q, want Z33500", jobs[0].Owner.String())
	}
}

func TestSetDataSpecs_SiteCommand(t *testing.T) {
	s, srv := dialMock(t)
	if err := s.SetDataSpecs(zftp.WithRecfmFB, zftp.WithLrecl(80), zftp.WithBlkSize(27920)); err != nil {
		t.Fatalf("SetDataSpecs: %v", err)
	}
	if !hasCmd(srv.Commands(), "SITE RECFM=FB LRECL=80 BLKSIZE=27920") {
		t.Errorf("expected SITE RECFM=FB LRECL=80 BLKSIZE=27920; got %v", srv.Commands())
	}
}

func TestSetDataSpecs_InvalidRejected(t *testing.T) {
	s, _ := dialMock(t)
	if err := s.SetDataSpecs(zftp.WithLrecl(0)); err == nil {
		t.Error("WithLrecl(0): want error")
	}
	if err := s.SetDataSpecs(); err == nil {
		t.Error("no attributes: want error")
	}
}
