package zftp

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v0/eol"
	"gopkg.in/ro-ag/zftp.v0/internal/helper"
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

// StatusUpdater interface to change attributes of the FTP session.
// Wrapper on SITE command
type StatusUpdater interface {
	// FileType Set the FILETYPE statement to specify the method of operation for FTP.
	FileType(Type string) error

	// JesEntryLimit Use the JESENTRYLIMIT statement to specify the number of entries that can be displayed concurrently
	// through a LIST or NLST command when FILETYPE=JES and JESINTERFACELEVEL=2.
	JesEntryLimit(limit int) error

	// JesGetByDSN Use the JESGETBYDSN statement to specify whether the JES interface uses the DSN or the job name to
	// retrieve the job output.
	JesGetByDSN(option bool) error

	// JesJobName Set the JESJOBNAME statement to specify the job name for FTP.
	JesJobName(expression string) error

	// JesLrecl Set the JESLRECL statement to specify the record length of the job being submitted.
	JesLrecl(len int) error

	// JesOwner Set the JESOWNER statement to specify the owner of the job for FTP.
	JesOwner(expression string) error

	// JesPutGetTimeOut Set the JESPUTGETTIMEOUT statement to specify the time-out value for the JES interface.
	JesPutGetTimeOut(seconds int) error

	// JesStatus Set the JESSTATUS statement to specify the status of the job for FTP.
	JesStatus(status string) error

	// ListLevel Set the LISTLEVEL statement to specify the level of the list command for FTP.
	ListLevel(level int) error

	// SBSendEol Indicates which end-of-line sequence to use when ENCODING is SBCS, the data is ASCII, and data is being sent to the client.
	SBSendEol(eol eol.LineBreaker) error

	// MBSendEol Indicates which end-of-line sequence to use when the ENCODING value is SBCS, the data is ASCII, and data is being sent to the server.
	MBSendEol(eol eol.LineBreaker) error
}

// SetStatusOf returns a StatusUpdater interface for the current FTP session.
func (s *FTPSession) SetStatusOf() StatusUpdater {
	return helper.SetFeature(s.Site)
}
