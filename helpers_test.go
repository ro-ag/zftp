// SPDX-License-Identifier: Apache-2.0

package zftp_test

import "strings"

// cmdIndex returns the index of the first received command line equal (case-
// and space-insensitively) to want, or -1.
func cmdIndex(cmds []string, want string) int {
	for i, c := range cmds {
		if strings.EqualFold(strings.TrimSpace(c), want) {
			return i
		}
	}
	return -1
}

// hasCmd reports whether want appears in the received command lines.
func hasCmd(cmds []string, want string) bool { return cmdIndex(cmds, want) >= 0 }
