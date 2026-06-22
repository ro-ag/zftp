// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"fmt"
	"strings"
)

// Site sends the SITE command to the FTP server and returns the raw response,
// translating z/OS "Unrecognized parameter" / "Parameter ignored" replies into
// errors.
func (s *FTPSession) Site(subCommand string, a ...string) (string, error) {
	args := strings.Join(a, " ")
	subCommand = strings.TrimSpace(strings.ToUpper(subCommand))
	subCommandWithArgs := fmt.Sprintf("%s %s", subCommand, args)
	str, err := s.SendCommand(CodeCmdOK, "SITE", subCommandWithArgs)
	lines := strings.Split(str, "\n")
	switch {
	case err != nil:
		return "", err
	case strings.Contains(str, "Unrecognized parameter"):
		return "", fmt.Errorf("error : '%s', %s", subCommandWithArgs, lines[0])
	case strings.Contains(str, "Parameter ignored"):
		return "", fmt.Errorf("error : '%s', %s", subCommandWithArgs, lines[0])
	default:
		return str, nil
	}
}

// SetStatusOf returns a *StatusSetter for changing z/OS session attributes (via
// SITE) on the current session. See StatusSetter for the available setters.
func (s *FTPSession) SetStatusOf() *StatusSetter {
	return &StatusSetter{site: s.Site}
}
