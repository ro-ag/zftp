// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"testing"

	zftp "gopkg.in/ro-ag/zftp.v2"
)

// TestStatusOf_ReturnsConcrete is a compile-time guarantee that the status
// accessors return concrete types, not interfaces (the core of the API cleanup).
func TestStatusOf_ReturnsConcrete(t *testing.T) {
	s, _ := dialMock(t)
	var _ *zftp.ServerStatus = s.StatusOf()
	var _ *zftp.StatusSetter = s.SetStatusOf()
}

func TestServerStatus_BlockSize(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("XSTA (BLOCKSIze",
		"211-Record format FB, Lrecl: 80, Blocksize: 27998",
		"211 *** end of status ***")

	bs, err := s.StatusOf().BlockSize()
	if err != nil {
		t.Fatalf("BlockSize: %v", err)
	}
	if bs != 27998 {
		t.Errorf("BlockSize = %d, want 27998", bs)
	}
}

func TestServerStatus_FileType(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("XSTA (FileType",
		"211-FileType JES (Job Entry Subsystem)",
		"211 *** end of status ***")

	ft, err := s.StatusOf().FileType()
	if err != nil {
		t.Fatalf("FileType: %v", err)
	}
	if ft != "JES" {
		t.Errorf("FileType = %q, want JES", ft)
	}
}

func TestStatusSetter_FileType(t *testing.T) {
	s, _ := dialMock(t)
	// Valid values reach the server (mock SITE -> 200).
	if err := s.SetStatusOf().FileType("SEQ"); err != nil {
		t.Errorf("FileType(SEQ): %v", err)
	}
	// Invalid values are rejected client-side before any SITE command.
	if err := s.SetStatusOf().FileType("BOGUS"); err == nil {
		t.Error("FileType(BOGUS): want error")
	}
}

func TestTransferType_Concrete(t *testing.T) {
	if !zftp.TypeAscii.IsAscii() || zftp.TypeAscii.IsBinary() {
		t.Error("TypeAscii classification wrong")
	}
	if !zftp.TypeBinary.IsBinary() || zftp.TypeBinary.IsAscii() {
		t.Error("TypeBinary classification wrong")
	}
	if zftp.TypeImage != zftp.TypeBinary {
		t.Error("TypeImage should equal TypeBinary")
	}
	if got := zftp.TypeAscii.Name(); got != "ASCII" {
		t.Errorf("TypeAscii.Name() = %q, want ASCII", got)
	}
}
