package zftp

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ro-ag/zftp.v0/hfs"
	"strings"
)

// List returns a list of files and directories in the current working directory
// it returns the raw lines by the server
func (s *FTPSession) anyList(cmd, expression string) ([]string, error) {
	cmd = strings.TrimSpace(strings.ToUpper(cmd))
	trimLine := false
	switch cmd {
	case "LIST", "MLSD":
		break
	case "NLST":
		trimLine = true
		break
	default:
		log.Panicf("invalid command: %s", cmd)
	}

	port, err := s.SetPassiveMode()
	if err != nil {
		return nil, err
	}

	child, err := s.newChildConnection(port)
	if err != nil {
		return nil, err
	}
	defer func(child *childConnection) {
		if err := child.Close(); err != nil {
			log.Error(err)
		}
	}(child)

	_, err = s.SendCommand(CodeListOK, cmd, expression)
	if err != nil {
		return nil, fmt.Errorf("error while sending list command: %s", err)
	}

	lines, err := make([]string, 0), error(nil)

	for child.Scanner().Scan() {
		if child.IsClosed() {
			break
		}
		line := child.Scanner().Text()
		if err := child.Scanner().Err(); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("error while scanning child connection: %s", err)
		}
		if trimLine {
			line = strings.TrimSpace(line)
		}
		lines = append(lines, line)
		log.Debugf("[res] %s", line)
	}

	err = child.Close()
	if err != nil {
		return nil, err
	}

	err = s.checkLast(CodeFileActionOK)
	if err != nil {
		return nil, fmt.Errorf("error while checking last response: %s", err)
	}
	return lines, nil
}

// List returns a list of files matching the given expression.
func (s *FTPSession) List(expression string) ([]string, error) {
	return s.anyList("LIST", expression)
}

// NList returns a plane list of files matching the given expression. It does not include file attributes.
func (s *FTPSession) NList(expression string) ([]string, error) {
	return s.anyList("NLST", expression)
}

// ListDatasets returns a list of files matching the given expression, including file attributes.
func (s *FTPSession) ListDatasets(expression string) ([]hfs.Dataset, error) {
	lines, err := s.List(expression)
	if err != nil {
		return nil, err
	}
	datasets := make([]hfs.Dataset, 0)
	for i := range lines {
		if i == 0 {
			continue
		}
		dataset, errParsing := hfs.ParseDataset(lines[i])
		if errParsing != nil {
			return nil, errParsing
		}
		datasets = append(datasets, dataset)
	}
	return datasets, nil
}

// ListPds returns a list of files matching the given expression, including file attributes.
func (s *FTPSession) ListPds(expression string) ([]hfs.PdsMember, error) {
	lines, err := s.List(expression)
	if err != nil {
		return nil, err
	}
	members := make([]hfs.PdsMember, 0)
	for i := range lines {
		if i == 0 {
			continue
		}
		member, errParsing := hfs.ParseMember(lines[i])
		if errParsing != nil {
			return nil, errParsing
		}
		members = append(members, member)
	}
	return members, nil
}
