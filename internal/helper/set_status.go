package helper

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v0/eol"
)

type SetFeature func(subCommand string, a ...string) (string, error)

// FileType Set the FILETYPE statement to specify the method of operation for FTP.
// Valid values are JES, SEQ, and SQL.
// ref: https://www.ibm.com/docs/en/zos/2.4.0?topic=protocol-filetype-ftp-client-server-statement
func (site SetFeature) FileType(Type string) error {
	switch Type {
	case "JES", "SEQ", "SQL":
		break
	default:
		return fmt.Errorf("error : '%s', %s", Type, "Unrecognized parameter")
	}

	// site is pointing to the Site function in site.go
	_, err := site(fmt.Sprintf("FILETYPE=%s", Type))

	return err
}

func (site SetFeature) JesJobName(expression string) error {
	_, err := site(fmt.Sprintf("JESJOBNAME=%s", expression))
	return err
}

func (site SetFeature) JesOwner(expression string) error {
	_, err := site(fmt.Sprintf("JESOWNER=%s", expression))
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
