// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"strings"
	"testing"
)

// fakeStatBlock is a neutral STAT reply modeling the real z/OS FTP "quote stat"
// 211- output format (see https://www.mslinn.com/mainframe/3000-run_jcl.html):
// a mix of "KEY is VALUE", "Server site variable KEY is set to VALUE", free-form
// prose, and a wrapped continuation line, terminated by "211 *** end of status".
const fakeStatBlock = "211-Server FTP talking to host 10.0.0.1, port 49884\n" +
	"211-User: MVSUSER  Working directory: MVSUSER.\n" +
	"211-Automatic recall of migrated data sets.\n" +
	"211-Inactivity timer is set to 300\n" +
	"211-Server site variable DSWAITTIMEREPLY is set to 60\n" +
	"211-Trailing blanks are removed from a fixed format data set when it is\n" +
	"211- retrieved.\n" +
	"211-FileType JES (MVS Job Spool). JES Name is JES2\n" +
	"211-JESLRECL is 80\n" +
	"211-JESINTERFACELEVEL is 1\n" +
	"211-UMASK value is 027\n" +
	"211-Server site variable LISTLEVEL is set to 0\n" +
	"211 *** end of status ***"

func TestServerStatus_Snapshot(t *testing.T) {
	calls := 0
	ss := &ServerStatus{stat: func(a ...string) (string, error) {
		calls++
		return fakeStatBlock, nil
	}}

	snap, err := ss.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if calls != 1 {
		t.Errorf("STAT issued %d times, want 1 round-trip", calls)
	}

	for _, c := range []struct{ key, want string }{
		{"JESLRECL", "80"},
		{"JESINTERFACELEVEL", "1"},
		{"UMASK value", "027"},
		{"DSWAITTIMEREPLY", "60"}, // "Server site variable " prefix stripped
		{"LISTLEVEL", "0"},        // ditto, " is set to " form
	} {
		if v, ok := snap.Get(c.key); !ok || v != c.want {
			t.Errorf("Get(%q) = %q,%v; want %q,true", c.key, v, ok, c.want)
		}
	}

	for _, ln := range snap.Lines() {
		if strings.Contains(ln, "end of status") {
			t.Errorf("Lines() still contains the terminator: %q", ln)
		}
	}

	var rejoined bool
	for _, ln := range snap.Lines() {
		if strings.Contains(ln, "Trailing blanks") && strings.Contains(ln, "retrieved.") {
			rejoined = true
		}
	}
	if !rejoined {
		t.Error("wrapped continuation line was not rejoined into a single Lines() entry")
	}
}
