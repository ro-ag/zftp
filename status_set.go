// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v2/eol"
	"strings"
)

// StatusSetter changes z/OS session attributes via the SITE command. It is
// returned by FTPSession.SetStatusOf; each method issues one SITE subcommand.
// Obtain it from the session rather than constructing it.
type StatusSetter struct {
	site func(subCommand string, a ...string) (string, error)
}

// FileType Set the FILETYPE statement to specify the method of operation for FTP.
// Valid values are JES, SEQ, and SQL.
// ref: https://www.ibm.com/docs/en/zos/2.4.0?topic=protocol-filetype-ftp-client-server-statement
func (s *StatusSetter) FileType(Type string) error {
	switch {
	case Type == "SEQ":
		break
	case strings.HasPrefix(Type, "SQL"):
		break
	case strings.HasPrefix(Type, "JES"):
		break
	default:
		return fmt.Errorf("error : '%s', %s", Type, "Unrecognized parameter")
	}

	// site is pointing to the Site function in site.go
	_, err := s.site(fmt.Sprintf("FILETYPE=%s", Type))

	return err
}

// JesEntryLimit sets the maximum number of JES entries a query may return (SITE
// JESENTRYLIMIT). The valid range is 0 to 1024.
func (s *StatusSetter) JesEntryLimit(limit int) error {
	if limit < 0 || limit > 1024 {
		return fmt.Errorf("JesEntryLimit must be between 0 and 1024")
	}

	_, err := s.site(fmt.Sprintf("JESENTRYLimit=%d", limit))
	return err
}

// JesGetByDSN enables or disables retrieving a job's output by DSN (SITE
// JESGETBYDSN / NOJESGETBYDSN).
func (s *StatusSetter) JesGetByDSN(option bool) error {
	cmd := "NOJESGETBYDSN"
	if option {
		cmd = "JESGETBYDSN"
	}
	_, err := s.site(cmd)

	return err
}

// JesJobName sets the job-name filter applied to JES queries (SITE JESJOBNAME);
// "*" matches every job name.
func (s *StatusSetter) JesJobName(expression string) error {
	_, err := s.site(fmt.Sprintf("JESJOBNAME=%s", expression))
	return err
}

// JesLrecl Use the JESLRECL statement to specify the record length of the jobs being submitted.
// The record length of the job being submitted. The valid range is 1 - 254. The default is 80. If you specify length as *, FTP uses the length value from the LRECL statement.
func (s *StatusSetter) JesLrecl(len int) error {
	if len < 1 || len > 254 {
		return fmt.Errorf("JesLrecl must be between 1 and 254")
	}

	_, err := s.site(fmt.Sprintf("JESLRECL=%d", len))
	return err
}

// JesRecfm Use the JESRECFM statement to specify the record format of jobs being submitted. This is the record format used during dynamic allocation of the internal reader when submitting jobs to JES
func (s *StatusSetter) JesRecfm(expression string) error {
	switch expression {
	case "F", "V", "*":
		break
	default:
		return fmt.Errorf("error : '%s', %s", expression, "Unrecognized parameter")
	}
	_, err := s.site(fmt.Sprintf("JesRecfm=%s", expression))
	return err
}

// JesPutGetTimeOut Use the JESPUTGETTO statement to specify the number of seconds of the JES PutGet timeout.
// The number of seconds of the JES PutGet timeout. The valid range is 0 - 86 400 (24 hours). The default is 600 (10 minutes).
func (s *StatusSetter) JesPutGetTimeOut(seconds int) error {
	if seconds < 0 || seconds > 86400 {
		return fmt.Errorf("JesPutGetTimeOut must be between 0 and 86400")
	}

	_, err := s.site(fmt.Sprintf("JESPUTGETTO=%d", seconds))
	return err
}

// JesOwner sets the owner filter applied to JES queries (SITE JESOWNER); "*"
// matches every owner.
func (s *StatusSetter) JesOwner(expression string) error {
	_, err := s.site(fmt.Sprintf("JESOWNER=%s", expression))
	return err
}

// JesStatus sets the job-status filter applied to JES queries (SITE JESSTATUS).
// Valid values are ALL, ACTIVE, OUTPUT, INPUT, EXECUTION, JOBLOG, JOBMSG, and
// JOBSTATUS.
func (s *StatusSetter) JesStatus(expression string) error {
	switch expression {
	case "ALL", "ACTIVE", "OUTPUT", "INPUT", "EXECUTION", "JOBLOG", "JOBMSG", "JOBSTATUS":
		break
	default:
		return fmt.Errorf("error : '%s', %s", expression, "Unrecognized parameter")
	}
	_, err := s.site(fmt.Sprintf("JESSTATUS=%s", expression))
	return err
}

// ListLevel sets the LISTLEVEL that controls the column layout of dataset
// listings (SITE LISTLEVEL).
func (s *StatusSetter) ListLevel(level int) error {
	_, err := s.site(fmt.Sprintf("LISTLEVEL=%d", level))
	return err
}

// SBSendEol sets the end-of-line sequence appended to outbound single-byte data
// (SITE SBSENDEOL).
func (s *StatusSetter) SBSendEol(eol eol.LineBreaker) error {
	_, err := s.site(fmt.Sprintf("SBSENDEOL=%s", eol.String()))
	return err
}

// MBSendEol sets the end-of-line sequence appended to outbound multibyte data
// (SITE MBSENDEOL).
func (s *StatusSetter) MBSendEol(eol eol.LineBreaker) error {
	_, err := s.site(fmt.Sprintf("MBSENDEOL=%s", eol.String()))
	return err
}
