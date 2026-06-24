// SPDX-License-Identifier: Apache-2.0

package hfs_test

import (
	"errors"
	"strings"
	"testing"

	"gopkg.in/ro-ag/zftp.v2/hfs"
)

// level2Detail builds a JesInterfaceLevel=2 InfoJobDetail from a single job line
// by prepending the column header the level is detected from.
func level2Detail(t *testing.T, jobLine string, rest ...string) (*hfs.InfoJobDetail, error) {
	t.Helper()
	records := append([]string{
		"JOBNAME  JOBID    OWNER    STATUS CLASS",
		jobLine,
	}, rest...)
	return hfs.ParseInfoJobDetail(records)
}

// TestParseInfoJob_Level1_SpoolCount verifies a JesInterfaceLevel=1 listing parses
// the trailing "N spool files" column into SpoolFiles and leaves Class empty,
// instead of dumping the spool-file text into Class.
func TestParseInfoJob_Level1_SpoolCount(t *testing.T) {
	jobs, err := hfs.ParseInfoJob([]string{"Z33552   TSU06321 OUTPUT  3 spool files"})
	if err != nil {
		t.Fatalf("ParseInfoJob: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("got %d jobs, want 1", len(jobs))
	}
	j := jobs[0]
	if j.SpoolFiles != 3 {
		t.Errorf("SpoolFiles = %d, want 3", j.SpoolFiles)
	}
	if got := j.Class.String(); got != "" {
		t.Errorf("Class = %q, want empty for a level-1 record", got)
	}
	if got := j.Status.String(); got != "OUTPUT" {
		t.Errorf("Status = %q, want OUTPUT", got)
	}
}

// TestReturnCode_AbendWithoutNumericCode verifies an ABEND with no parseable
// "ABEND=NNN" code (e.g. "ABEND S0C4") still reports ErrAbendedJob instead of a
// generic "no return code found" — the abend must not be dropped.
func TestReturnCode_AbendWithoutNumericCode(t *testing.T) {
	jd, err := level2Detail(t, "MYJOB    JOB00009 Z00000   OUTPUT A        ABEND S0C4")
	if err != nil {
		t.Fatalf("ParseInfoJobDetail: %v", err)
	}
	if _, err := jd.ReturnCode(); !errors.Is(err, hfs.ErrAbendedJob) {
		t.Errorf("ReturnCode() err = %v, want ErrAbendedJob", err)
	}
}

// TestAbendCode_RealCodes verifies the real z/OS abend completion codes are
// captured and exposed. Real abend codes are alphanumeric (system Scde, user
// Ucde), not decimal — e.g. S0C4, S806, U0778 — so they are surfaced as a string
// via AbendCode while ReturnCode reports (-1, ErrAbendedJob). Formats per IBM
// message IEF450I "... - ABEND=Scde Ucde ..." and the JES listing status "ABEND
// Scde" (modeled here with neutral job names).
func TestAbendCode_RealCodes(t *testing.T) {
	cases := []struct{ record, code string }{
		{"MYJOB    JOB00001 Z00000   OUTPUT A        ABEND=S806", "S806"},
		{"MYJOB    JOB00002 Z00000   OUTPUT A        ABEND S0C4", "S0C4"},
		{"MYJOB    JOB00003 Z00000   OUTPUT A        ABEND=U0778", "U0778"},
	}
	for _, c := range cases {
		jd, err := level2Detail(t, c.record)
		if err != nil {
			t.Fatalf("ParseInfoJobDetail(%q): %v", c.record, err)
		}
		if code, ok := jd.AbendCode(); !ok || code != c.code {
			t.Errorf("AbendCode(%q) = %q,%v; want %q,true", c.record, code, ok, c.code)
		}
		if rc, err := jd.ReturnCode(); !errors.Is(err, hfs.ErrAbendedJob) || rc != -1 {
			t.Errorf("ReturnCode(%q) = %d,%v; want -1,ErrAbendedJob", c.record, rc, err)
		}
	}
	jd, err := level2Detail(t, "MYJOB    JOB00004 Z00000   OUTPUT A        RC=0004")
	if err != nil {
		t.Fatal(err)
	}
	if code, ok := jd.AbendCode(); ok {
		t.Errorf("non-abend AbendCode = %q,true; want empty,false", code)
	}
	if rc, err := jd.ReturnCode(); err != nil || rc != 4 {
		t.Errorf("RC job ReturnCode = %d,%v; want 4,nil", rc, err)
	}
}

// TestParseInfoJobDetail_StrayBlankLine verifies a stray blank line in the detail
// block (here between the job record and the "--------" separator) does not abort
// parsing: the separator and column header are located by content, not fixed
// offsets.
func TestParseInfoJobDetail_StrayBlankLine(t *testing.T) {
	records := []string{
		"JOBNAME  JOBID    OWNER    STATUS CLASS",
		"ANOTHER  JOB06184 Z33500   OUTPUT A        RC=0000",
		"", // stray blank line
		"--------",
		"         ID  STEPNAME PROCSTEP C DDNAME   BYTE-COUNT",
		"         001 JES2        N/A   A JESMSGLG      1234",
		"         002 STEP2       N/A   A SYSPRINT       251",
		"2 spool files",
	}
	jd, err := hfs.ParseInfoJobDetail(records)
	if err != nil {
		t.Fatalf("ParseInfoJobDetail with stray blank: %v", err)
	}
	if got := len(jd.Detail()); got != 2 {
		t.Errorf("detail count = %d, want 2", got)
	}
}

// TestParseInfoJob_ErrorReportsOriginalLine verifies a parse error reports the
// line number from the ORIGINAL input, not the index after blank lines have been
// filtered out.
func TestParseInfoJob_ErrorReportsOriginalLine(t *testing.T) {
	records := []string{
		"", // line 1
		"", // line 2
		"GOODJOB  JOB00001 OUTPUT  3 spool files", // line 3
		"BADRECORD", // line 4 (malformed)
	}
	_, err := hfs.ParseInfoJob(records)
	if err == nil {
		t.Fatal("want error for malformed record")
	}
	if !strings.Contains(err.Error(), "line 4") {
		t.Errorf("error should report original input line 4, got: %v", err)
	}
}
