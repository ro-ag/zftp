// SPDX-License-Identifier: Apache-2.0
package zftp_test

import (
	"errors"
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
