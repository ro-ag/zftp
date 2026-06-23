// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"testing"

	zftp "gopkg.in/ro-ag/zftp.v2"
)

// TestWithJesJobPattern_GroupValidation verifies WithJesJobPattern accepts a
// pattern with exactly one capturing group (the job-id) and rejects patterns with
// none or more than one, as well as patterns that do not compile. The job-id is
// read from submatch[1], so a wrong group count would silently break extraction.
func TestWithJesJobPattern_GroupValidation(t *testing.T) {
	s, _ := dialMock(t)

	if err := zftp.WithJesJobPattern(`(JOB\d{5})`).Apply(s); err != nil {
		t.Errorf("one capturing group: unexpected error: %v", err)
	}
	if err := zftp.WithJesJobPattern(`JOB\d{5}`).Apply(s); err == nil {
		t.Error("zero capturing groups: want error")
	}
	if err := zftp.WithJesJobPattern(`(JOB)(\d{5})`).Apply(s); err == nil {
		t.Error("two capturing groups: want error")
	}
	if err := zftp.WithJesJobPattern(`(`).Apply(s); err == nil {
		t.Error("invalid regexp: want error")
	}
}
