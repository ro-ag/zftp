// SPDX-License-Identifier: Apache-2.0

package hfs_test

import (
	"testing"

	"gopkg.in/ro-ag/zftp.v2/hfs"
)

// InfoJobDetail.Detail must return a slice of an EXPORTED type so external
// callers can name it, declare variables of it, and read its fields. Returning
// []jobDetail (unexported) makes per-step JES detail unusable outside the package.
func TestDetailReturnsExportedType(t *testing.T) {
	var jd hfs.InfoJobDetail
	var _ []hfs.JobDetail = jd.Detail()
}
