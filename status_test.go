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

func TestServerStatus_Lrecl(t *testing.T) {
	s, srv := dialMock(t)
	// Distinct Lrecl (80) and Blocksize (27920) so a getter that returns the
	// wrong capture group of the shared recFmt regex is visible. A real z/OS
	// XSTA status reply is multiline (211-... continuation, 211 terminator).
	const recLine = "211-Record format FB, Lrecl: 80, Blocksize: 27920"
	srv.Script("XSTA (Lrecl", recLine, "211 *** end of status ***")
	srv.Script("XSTA (BLOCKSIze", recLine, "211 *** end of status ***")

	lrecl, err := s.StatusOf().Lrecl()
	if err != nil {
		t.Fatalf("Lrecl: %v", err)
	}
	if lrecl != 80 {
		t.Errorf("Lrecl = %d, want 80 (returning the Blocksize is the bug)", lrecl)
	}

	// Guard against a "fix" that merely swaps the two capture groups: BlockSize
	// must still report the Blocksize from the same reply shape.
	bs, err := s.StatusOf().BlockSize()
	if err != nil {
		t.Fatalf("BlockSize: %v", err)
	}
	if bs != 27920 {
		t.Errorf("BlockSize = %d, want 27920", bs)
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

// TestServerStatus_DCBDSN_DottedName verifies a fully-qualified, dotted model DSN
// survives parsing. The old LastWord regex `\s(\w+)$` excluded '.', so a dotted
// DCBDSN came back empty.
func TestServerStatus_DCBDSN_DottedName(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("XSTA (DCBDSN",
		"211-DCBDSN MY.MODEL.DSN",
		"211 *** end of status ***")

	got, err := s.StatusOf().DCBDSN()
	if err != nil {
		t.Fatalf("DCBDSN: %v", err)
	}
	if got != "MY.MODEL.DSN" {
		t.Errorf("DCBDSN = %q, want MY.MODEL.DSN", got)
	}
}

// TestServerStatus_UMask_Octal verifies UMask is parsed as octal: "022" is octal
// for decimal 18. It was previously parsed as decimal (returning 22).
func TestServerStatus_UMask_Octal(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("XSTA (UMask",
		"211-UMask 022",
		"211 *** end of status ***")

	got, err := s.StatusOf().UMask()
	if err != nil {
		t.Fatalf("UMask: %v", err)
	}
	if got != 18 {
		t.Errorf("UMask = %d, want 18 (octal 022)", got)
	}
}

// TestServerStatus_SBDataConn_Codepage verifies SBDataConn returns the codepage
// as a string. SBDATACONN is a codepage (e.g. IBM-1047), not an integer; the old
// (int) signature could not represent it and the hyphenated token failed the
// integer parse. The scripted reply is representative, not a captured LPAR reply.
func TestServerStatus_SBDataConn_Codepage(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("XSTA (SBDataConn",
		"211-SBDataConn IBM-1047",
		"211 *** end of status ***")

	got, err := s.StatusOf().SBDataConn()
	if err != nil {
		t.Fatalf("SBDataConn: %v", err)
	}
	if got != "IBM-1047" {
		t.Errorf("SBDataConn = %q, want IBM-1047", got)
	}
}

// TestServerStatus_Unit_Sibling is a regression guard: a plain-token LastWord
// sibling getter must keep returning its token after the LastWord change.
func TestServerStatus_Unit_Sibling(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("XSTA (Unit",
		"211-Unit SYSDA",
		"211 *** end of status ***")

	got, err := s.StatusOf().Unit()
	if err != nil {
		t.Fatalf("Unit: %v", err)
	}
	if got != "SYSDA" {
		t.Errorf("Unit = %q, want SYSDA", got)
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
