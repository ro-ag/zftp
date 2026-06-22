// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"fmt"
	"strings"
)

// Stat returns the server status string.
func (s *FTPSession) Stat(a ...string) (string, error) {
	return s.SendCommand(CodeSysStatus, "STAT", a...)
}

// XStat issues an XSTA command to retrieve an individual status variable or
// property from the server's current status.
func (s *FTPSession) XStat(feature string) (string, error) {
	out, err := s.SendCommand(CodeSysStatus, "XSTA", fmt.Sprintf("(%s", feature))
	if err != nil {
		return "", err
	}

	out = strings.ReplaceAll(out, "*** end of status ***", "")
	out = strings.TrimSpace(out)
	return out, nil
}

// StatusOf returns a *ServerStatus for querying individual server status values
// (via XSTA) on the current session. See ServerStatus for the available getters.
//
// Reference: https://www.ibm.com/docs/en/zos/2.2.0?topic=fs-status-subcommand-retrieve-status-information-from-remote-host
func (s *FTPSession) StatusOf() *ServerStatus {
	return &ServerStatus{xstat: s.XStat}
}
