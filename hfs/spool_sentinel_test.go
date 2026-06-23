// SPDX-License-Identifier: Apache-2.0

package hfs_test

import (
	"errors"
	"testing"

	"gopkg.in/ro-ag/zftp.v2/hfs"
)

// TestReturnCode_SentinelsMatchable verifies the hfs sentinels are
// errors.Is-matchable straight from their producers: an ACTIVE job yields
// ErrActiveJob (returned by ParseInfoJobDetail) and a "JCL error" class yields
// ErrJCLError (returned by ReturnCode). ErrAbendedJob is covered by the ABEND
// ReturnCode tests. Records use the JesInterfaceLevel=2 shape (the level that
// carries Status/Class), detected from the JOBNAME column header.
func TestReturnCode_SentinelsMatchable(t *testing.T) {
	const header = "JOBNAME  JOBID    OWNER    STATUS CLASS"

	if _, err := hfs.ParseInfoJobDetail([]string{
		header,
		"MYJOB    JOB00001 Z00000   ACTIVE A",
	}); !errors.Is(err, hfs.ErrActiveJob) {
		t.Errorf("ACTIVE job: err = %v, want ErrActiveJob", err)
	}

	jd, err := hfs.ParseInfoJobDetail([]string{
		header,
		"MYJOB    JOB00002 Z00000   OUTPUT JCL error",
	})
	if err != nil {
		t.Fatalf("ParseInfoJobDetail: %v", err)
	}
	if _, err := jd.ReturnCode(); !errors.Is(err, hfs.ErrJCLError) {
		t.Errorf("JCL-error class: err = %v, want ErrJCLError", err)
	}
}
