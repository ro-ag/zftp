// SPDX-License-Identifier: Apache-2.0
package zftp_test

import (
	"errors"
	"sync"
	"testing"

	"gopkg.in/ro-ag/zftp.v2"
)

func TestDelete_OK(t *testing.T) {
	s, _ := dialMock(t)
	if err := s.Delete("'USER.OLD.DATA'"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestDelete_ServerError(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("DELE", "550 dataset not found")
	err := s.Delete("'USER.NOPE'")
	if !errors.Is(err, zftp.CodeError(550)) {
		t.Fatalf("Delete err = %v, want CodeError(550)", err)
	}
}

func TestMkdir_OK(t *testing.T) {
	s, _ := dialMock(t)
	if err := s.Mkdir("/u/user/newdir"); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
}

func TestMkdir_ServerError(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("MKD", "550 permission denied")
	err := s.Mkdir("/u/user/newdir")
	if !errors.Is(err, zftp.CodeError(550)) {
		t.Fatalf("Mkdir err = %v, want CodeError(550)", err)
	}
}

func TestRename_OK(t *testing.T) {
	s, srv := dialMock(t)
	if err := s.Rename("'USER.A'", "'USER.B'"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	cmds := srv.Commands()
	var sawFR, sawTO bool
	for i, c := range cmds {
		if c == "RNFR 'USER.A'" {
			sawFR = true
			if i+1 >= len(cmds) || cmds[i+1] != "RNTO 'USER.B'" {
				t.Fatalf("RNTO must immediately follow RNFR; got %v", cmds)
			}
		}
		if c == "RNTO 'USER.B'" {
			sawTO = true
		}
	}
	if !sawFR || !sawTO {
		t.Fatalf("missing RNFR/RNTO in %v", cmds)
	}
}

func TestRename_RNFRError(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("RNFR", "550 source not found")
	if err := s.Rename("'USER.NOPE'", "'USER.B'"); !errors.Is(err, zftp.CodeError(550)) {
		t.Fatalf("Rename err = %v, want CodeError(550)", err)
	}
}

// TestRename_NoInterleave runs many concurrent Renames + SendCommands under -race
// and asserts every RNTO immediately follows its RNFR (the pair never splits).
func TestRename_NoInterleave(t *testing.T) {
	s, srv := dialMock(t)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); _ = s.Rename("'A'", "'B'") }()
		go func() { defer wg.Done(); _, _ = s.SendCommand(zftp.CodeCmdOK, "NOOP") }()
	}
	wg.Wait()
	cmds := srv.Commands()
	for i, c := range cmds {
		if c == "RNFR 'A'" {
			if i+1 >= len(cmds) || cmds[i+1] != "RNTO 'B'" {
				t.Fatalf("RNFR/RNTO split by a concurrent command at %d: %v", i, cmds)
			}
		}
	}
}

func TestChmod_OK(t *testing.T) {
	s, srv := dialMock(t)
	if err := s.Chmod("750", "/u/user/file"); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	cmds := srv.Commands()
	want := "SITE CHMOD 750 /u/user/file"
	for _, c := range cmds {
		if c == want {
			return
		}
	}
	t.Fatalf("missing %q in %v", want, cmds)
}

func TestChmod_ServerError(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("SITE", "550 not permitted")
	if err := s.Chmod("750", "/u/user/file"); !errors.Is(err, zftp.CodeError(550)) {
		t.Fatalf("Chmod err = %v, want CodeError(550)", err)
	}
}
