package zftp

import (
	"fmt"
	"strings"
)

// Site sends the SITE command to the FTP server.
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

// SetFeatures interface to change attributes of the FTP session.
// Wrapper on SITE command
type SetFeatures interface {
	SetFileTypes(Type string) error
}
