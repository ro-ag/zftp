// SPDX-License-Identifier: Apache-2.0

package hfs_test

import (
	"errors"
	"testing"

	"gopkg.in/ro-ag/zftp.v2/hfs"
)

// detailFromLine builds an InfoJobDetail from a single JesInterfaceLevel=1 record
// line (Name JobId Status Class) via the public parser.
func detailFromLine(t *testing.T, line string) *hfs.InfoJobDetail {
	t.Helper()
	d, err := hfs.ParseInfoJobDetail([]string{line})
	if err != nil {
		t.Fatalf("ParseInfoJobDetail(%q): %v", line, err)
	}
	return d
}

// TestReturnCode_SentinelsMatchable verifies the hfs sentinels are
// errors.Is-matchable straight from their producer (ReturnCode): an ACTIVE job
// yields ErrActiveJob and a JCL-error class yields ErrJCLError. (ErrAbendedJob is
// covered by the ABEND cases in the ReturnCode robustness tests.)
func TestReturnCode_SentinelsMatchable(t *testing.T) {
	if _, err := detailFromLine(t, "MYJOB JOB00001 ACTIVE A").ReturnCode(); !errors.Is(err, hfs.ErrActiveJob) {
		t.Errorf("ACTIVE job: err = %v, want ErrActiveJob", err)
	}
	if _, err := detailFromLine(t, "MYJOB JOB00002 OUTPUT JCL error").ReturnCode(); !errors.Is(err, hfs.ErrJCLError) {
		t.Errorf("JCL-error class: err = %v, want ErrJCLError", err)
	}
}
