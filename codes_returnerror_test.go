// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"errors"
	"io"
	"testing"
)

// TestReturnError_IsByCode verifies a *ReturnError matches, under errors.Is,
// another *ReturnError carrying the same return code and not one with a different
// code — letting callers match by code without unpacking the error.
func TestReturnError_IsByCode(t *testing.T) {
	err := error(&ReturnError{rc: 550, wantRc: 250, message: "dataset not found"})

	if !errors.Is(err, &ReturnError{rc: 550}) {
		t.Errorf("errors.Is(err, &ReturnError{rc:550}) = false, want true")
	}
	if errors.Is(err, &ReturnError{rc: 551}) {
		t.Errorf("errors.Is(err, &ReturnError{rc:551}) = true, want false")
	}
}

// TestReturnError_UnwrapCause verifies an attached transport cause is reachable
// via errors.Is/errors.As through ReturnError.Unwrap, while the code match still
// works — a single error can be both "code 426" and "the connection failed".
func TestReturnError_UnwrapCause(t *testing.T) {
	err := error(&ReturnError{rc: 426, wantRc: 226, cause: io.ErrUnexpectedEOF})

	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("errors.Is(err, io.ErrUnexpectedEOF) = false, want true (Unwrap cause)")
	}
	if !errors.Is(err, &ReturnError{rc: 426}) {
		t.Errorf("code match lost when a cause is attached")
	}

	var re *ReturnError
	if !errors.As(err, &re) || re.ReturnCode() != 426 {
		t.Errorf("errors.As to *ReturnError failed; got %v", re)
	}
}

// TestCodeError_TargetByCode verifies the exported CodeError builds an errors.Is
// target so external callers (which cannot construct a *ReturnError with its
// unexported fields) can still match by code.
func TestCodeError_TargetByCode(t *testing.T) {
	err := error(&ReturnError{rc: 550, wantRc: 250})

	if !errors.Is(err, CodeError(CodeFileActionNotTakenPerm)) { // 550
		t.Errorf("errors.Is(err, CodeError(550)) = false, want true")
	}
	if errors.Is(err, CodeError(CodeBadFileName)) { // 553
		t.Errorf("errors.Is(err, CodeError(553)) = true, want false")
	}
}

// TestReturnError_NoCauseUnwrapNil documents that a pure protocol error has no
// cause: Unwrap returns nil so errors.Is does not match an unrelated transport
// error.
func TestReturnError_NoCauseUnwrapNil(t *testing.T) {
	err := error(&ReturnError{rc: 550, wantRc: 250})
	if errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("a causeless ReturnError must not match io.ErrUnexpectedEOF")
	}
	if u := errors.Unwrap(err); u != nil {
		t.Errorf("Unwrap = %v, want nil", u)
	}
}
