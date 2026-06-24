// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"errors"
	"testing"
)

// TestClassifyJesOutput verifies the JES output classifier — the producer of the
// JES sentinels — returns each sentinel errors.Is-matchably: an ABEND (ABAxxx)
// message yields ErrAba, an allocation/JCL (IEFxxx) message yields ErrIEF, both
// together yield ErrIEFAndABA, and clean output yields no error.
func TestClassifyJesOutput(t *testing.T) {
	if _, errType := classifyJesOutput("preamble\nABA001I task abended\ntail"); !errors.Is(errType, ErrAba) {
		t.Errorf("ABA-only: errType = %v, want ErrAba", errType)
	}
	if _, errType := classifyJesOutput("IEF001I write to message file failed"); !errors.Is(errType, ErrIEF) {
		t.Errorf("IEF-only: errType = %v, want ErrIEF", errType)
	}
	if details, errType := classifyJesOutput("ABA001I boom\nIEF001I oops"); !errors.Is(errType, ErrIEFAndABA) {
		t.Errorf("both: errType = %v (details %v), want ErrIEFAndABA", errType, details)
	}
	if details, errType := classifyJesOutput("all good\njob completed cleanly"); errType != nil || len(details) != 0 {
		t.Errorf("clean: errType=%v details=%v, want nil/empty", errType, details)
	}
}

// TestClassifyJesOutput_Abend verifies a real abend completion line is classified
// as ErrAbend (distinct from a generic IEF allocation message). Real abend codes
// are alphanumeric (system S0C4/S806, user U0778); the abend usually arrives as an
// IEF450I/IEF472I line, so ErrAbend takes precedence over ErrIEF. Format per IBM
// message IEF450I "... - ABEND=Scde Ucde ...".
func TestClassifyJesOutput_Abend(t *testing.T) {
	details, errType := classifyJesOutput("preamble\nIEF450I MVS001 STEP1 - ABEND=S0C4 REASON=00000004\ntail")
	if !errors.Is(errType, ErrAbend) {
		t.Errorf("system abend: errType = %v, want ErrAbend", errType)
	}
	if len(details) == 0 {
		t.Error("want the abend line captured in details")
	}
	if _, e := classifyJesOutput("$HASP395 MVS001 ENDED ABEND U0778"); !errors.Is(e, ErrAbend) {
		t.Errorf("user abend: errType = %v, want ErrAbend", e)
	}
	// A plain "ABEND" word with no completion code is not an abend classification.
	if _, e := classifyJesOutput("the ABEND handling routine ran"); errors.Is(e, ErrAbend) {
		t.Error("a bare ABEND word (no code) must not classify as ErrAbend")
	}
}
