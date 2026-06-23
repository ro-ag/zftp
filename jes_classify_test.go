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
	if errType, _ := classifyJesOutput("preamble\nABA001I task abended\ntail"); !errors.Is(errType, ErrAba) {
		t.Errorf("ABA-only: errType = %v, want ErrAba", errType)
	}
	if errType, _ := classifyJesOutput("IEF001I write to message file failed"); !errors.Is(errType, ErrIEF) {
		t.Errorf("IEF-only: errType = %v, want ErrIEF", errType)
	}
	if errType, details := classifyJesOutput("ABA001I boom\nIEF001I oops"); !errors.Is(errType, ErrIEFAndABA) {
		t.Errorf("both: errType = %v (details %v), want ErrIEFAndABA", errType, details)
	}
	if errType, details := classifyJesOutput("all good\njob completed cleanly"); errType != nil || len(details) != 0 {
		t.Errorf("clean: errType=%v details=%v, want nil/empty", errType, details)
	}
}
