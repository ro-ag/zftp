package helper

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v1/eol"
	"strings"
)

type SetFeature func(subCommand string, a ...string) (string, error)

// FileType Set the FILETYPE statement to specify the method of operation for FTP.
// Valid values are JES, SEQ, and SQL.
// ref: https://www.ibm.com/docs/en/zos/2.4.0?topic=protocol-filetype-ftp-client-server-statement
func (site SetFeature) FileType(Type string) error {
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
	_, err := site(fmt.Sprintf("FILETYPE=%s", Type))

	return err
}

func (site SetFeature) JesEntryLimit(limit int) error {
	if limit < 0 || limit > 1024 {
		return fmt.Errorf("JesEntryLimit must be between 0 and 1024")
	}

	_, err := site(fmt.Sprintf("JESENTRYLimit=%d", limit))
	return err
}

func (site SetFeature) JesGetByDSN(option bool) error {
	cmd := "NOJESGETBYDSN"
	if option {
		cmd = "JESGETBYDSN"
	}
	_, err := site(cmd)

	return err
}

func (site SetFeature) JesJobName(expression string) error {
	_, err := site(fmt.Sprintf("JESJOBNAME=%s", expression))
	return err
}

// JesLrecl Use the JESLRECL statement to specify the record length of the jobs being submitted.
// The record length of the job being submitted. The valid range is 1 - 254. The default is 80. If you specify length as *, FTP uses the length value from the LRECL statement.
func (site SetFeature) JesLrecl(len int) error {
	if len < 1 || len > 254 {
		return fmt.Errorf("JesLrecl must be between 1 and 254")
	}

	_, err := site(fmt.Sprintf("JESLRECL=%d", len))
	return err
}

// JesRecfm Use the JESRECFM statement to specify the record format of jobs being submitted. This is the record format used during dynamic allocation of the internal reader when submitting jobs to JES
func (site SetFeature) JesRecfm(expression string) error {
	switch expression {
	case "F", "V", "*":
		break
	default:
		return fmt.Errorf("error : '%s', %s", expression, "Unrecognized parameter")
	}
	_, err := site(fmt.Sprintf("JesRecfm=%s", expression))
	return err
}

// JesPutGetTimeOut Use the JESPUTGETTO statement to specify the number of seconds of the JES PutGet timeout.
// The number of seconds of the JES PutGet timeout. The valid range is 0 - 86 400 (24 hours). The default is 600 (10 minutes).
func (site SetFeature) JesPutGetTimeOut(seconds int) error {
	if seconds < 0 || seconds > 86400 {
		return fmt.Errorf("JesPutGetTimeOut must be between 0 and 86400")
	}

	_, err := site(fmt.Sprintf("JESPUTGETTO=%d", seconds))
	return err
}

func (site SetFeature) JesOwner(expression string) error {
	_, err := site(fmt.Sprintf("JESOWNER=%s", expression))
	return err
}

func (site SetFeature) JesStatus(expression string) error {
	switch expression {
	case "ALL", "ACTIVE", "OUTPUT", "INPUT", "EXECUTION", "JOBLOG", "JOBMSG", "JOBSTATUS":
		break
	default:
		return fmt.Errorf("error : '%s', %s", expression, "Unrecognized parameter")
	}
	_, err := site(fmt.Sprintf("JESSTATUS=%s", expression))
	return err
}

func (site SetFeature) ListLevel(level int) error {
	_, err := site(fmt.Sprintf("LISTLEVEL=%d", level))
	return err
}

func (site SetFeature) SBSendEol(eol eol.LineBreaker) error {
	_, err := site(fmt.Sprintf("SBSENDEOL=%s", eol.String()))
	return err
}

func (site SetFeature) MBSendEol(eol eol.LineBreaker) error {
	_, err := site(fmt.Sprintf("MBSENDEOL=%s", eol.String()))
	return err
}
