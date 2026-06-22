// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"context"
	"fmt"
	"strings"
)

// Site sends the SITE command to the FTP server and returns the raw response,
// translating z/OS "Unrecognized parameter" / "Parameter ignored" replies into
// errors.
func (s *FTPSession) Site(subCommand string, a ...string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.siteLocked(subCommand, a...)
}

// siteLocked issues a single SITE subcommand and interprets z/OS rejection
// replies. The caller must hold s.mu.
func (s *FTPSession) siteLocked(subCommand string, a ...string) (string, error) {
	args := strings.Join(a, " ")
	subCommand = strings.TrimSpace(strings.ToUpper(subCommand))
	subCommandWithArgs := fmt.Sprintf("%s %s", subCommand, args)
	str, err := s.sendLocked(context.Background(), CodeCmdOK, "SITE", subCommandWithArgs)
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

// setStatusOfLocked is like SetStatusOf but its setters assume s.mu is already
// held. It is used by methods that run a whole sequence under the lock, such as
// Login, where calling the public (locking) Site would deadlock.
func (s *FTPSession) setStatusOfLocked() *StatusSetter {
	return &StatusSetter{site: s.siteLocked}
}
