package zftp

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ro-ag/zftp.v0/hfs"
	"gopkg.in/ro-ag/zftp.v0/internal/utils"
	"strings"
)

// List returns a list of files and directories in the current working directory
// it returns the raw lines by the server, the list command response
func (s *FTPSession) anyList(cmd, expression string) ([]string, string, error) {

	cmd = strings.TrimSpace(strings.ToUpper(cmd))
	trimLine := false
	switch cmd {
	case "LIST":
		break
	case "NLST":
		trimLine = true
		break
	default:
		log.Panicf("invalid command: %s", cmd)
	}

	current := s.currType

	if current != TypeAscii {
		if err := s.SetType(TypeAscii); err != nil {
			return nil, "", err
		}
		defer func() {
			if err := s.SetType(current); err != nil {
				log.Error(err)
			}
		}()
	}

	port, err := s.SetPassiveMode()
	if err != nil {
		return nil, "", err
	}

	child, err := s.newChildConnection(port)
	if err != nil {
		return nil, "", err
	}
	defer func(child *childConnection) {
		if err := child.Close(); err != nil {
			log.Error(err)
		}
	}(child)

	resp, err := s.SendCommand(CodeListOK, cmd, expression)
	if err != nil {
		return nil, resp, fmt.Errorf("error while sending list command: %s", err)
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
			return nil, resp, fmt.Errorf("error while scanning child connection: %s", err)
		}
		if trimLine {
			line = strings.TrimSpace(line)
		}
		lines = append(lines, line)
		log.Debugf("[psv] %s", line)
	}

	err = child.Close()
	if err != nil {
		return nil, resp, err
	}

	_, err = s.checkLast(CodeFileActionOK)
	if err != nil {
		return nil, resp, fmt.Errorf("error while checking last response: %s", err)
	}
	return lines, resp, nil
}

// List returns a list of files matching the given expression.
func (s *FTPSession) List(expression string) ([]string, error) {
	lines, _, err := s.anyList("LIST", expression)
	return lines, err
}

// NList returns a plane list of files matching the given expression. It does not include file attributes.
func (s *FTPSession) NList(expression string) ([]string, error) {
	lines, _, err := s.anyList("NLST", expression)
	return lines, err
}

// ListDatasets returns a list of files matching the given expression, including file attributes.
func (s *FTPSession) ListDatasets(expression string) ([]hfs.InfoDataset, error) {

	curr, err := utils.SetValueAndGetCurrent("SEQ", s.SetStatusOf().FileType, s.StatusOf().FileType)
	if err != nil {
		return nil, err
	}
	defer curr.Restore()

	lines, err := s.List(expression)
	if err != nil {
		return nil, err
	}
	datasets := make([]hfs.InfoDataset, 0)
	for i := range lines {
		if i == 0 {
			continue
		}
		dataset, errParsing := hfs.ParseInfoDataset(lines[i])
		if errParsing != nil {
			return nil, fmt.Errorf("error while parsing record \"%s\": %s", lines[i], errParsing)
		}
		datasets = append(datasets, dataset)
	}
	return datasets, nil
}

// ListPds returns a list of files matching the given expression, including file attributes.
func (s *FTPSession) ListPds(expression string) ([]hfs.InfoPdsMember, error) {

	curr, err := utils.SetValueAndGetCurrent("SEQ", s.SetStatusOf().FileType, s.StatusOf().FileType)
	if err != nil {
		return nil, err
	}
	defer curr.Restore()

	lines, err := s.List(expression)
	if err != nil {
		return nil, err
	}
	members := make([]hfs.InfoPdsMember, 0)
	for i := range lines {
		if i == 0 {
			continue
		}
		member, errParsing := hfs.ParseInfoPdsMember(lines[i])
		if errParsing != nil {
			return nil, errParsing
		}
		members = append(members, member)
	}
	return members, nil
}

// ListSpool list jobs in the spool
func (s *FTPSession) ListSpool(expression string) ([]hfs.InfoJob, error) {

	expression = strings.TrimSpace(expression)
	if expression == "" {
		expression = "*"
	}
	if !utils.RegexSearchPattern.MatchString(expression) {
		return nil, fmt.Errorf("invalid search pattern: %s", expression)
	}

	curr, err := utils.SetValueAndGetCurrent("JES", s.SetStatusOf().FileType, s.StatusOf().FileType)
	if err != nil {
		return nil, err
	}
	defer curr.Restore()

	lines, err := s.List(expression)
	if err != nil {
		return nil, fmt.Errorf("failed to list spool jobs: %w", err)
	}

	jobs, err := hfs.ParseInfoJob(lines)
	if err != nil {
		return nil, fmt.Errorf("failed to parse spool job status: %w", err)
	}

	return jobs, nil
}
