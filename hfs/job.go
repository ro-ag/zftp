package hfs

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	whitespaceRegex    = regexp.MustCompile(`\s+`)
	searchPatternRegex = regexp.MustCompile(`[*?]|^\s*$`)
)

const (
	level1 = iota + 4
	level2
	spool
)

type JobStatus struct {
	Name   StringField `json:"Name"`
	JobId  StringField `json:"JobId"`
	Owner  StringField `json:"Owner"`
	Status StringField `json:"Status"`
	Class  StringField `json:"Class"`
	detail []JobSpool
}

type JobSpool struct {
	Id        IntField    `json:"Id"`
	StepName  StringField `json:"StepName"`
	ProcSpec  StringField `json:"ProcSpec"`
	C         StringField `json:"C"`
	DDName    StringField `json:"DDName"`
	ByteCount IntField    `json:"ByteCount"`
}

func ParseJobStatus(records []string, expression string) ([]JobStatus, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("no records provided")
	}
	expression = strings.TrimSpace(expression)

	var (
		isTargetJob = !searchPatternRegex.MatchString(expression)
		jobs        = make([]JobStatus, 0)
		spools      = make([]JobSpool, 0)
		kind        = level1
	)

	if strings.Contains(records[0], "JOBNAME") {
		kind = level2
	}

	for i, record := range records {

		if i == 0 && kind == level2 {
			continue
		}
		if strings.HasPrefix(record, "---") || strings.HasPrefix(record, "STEP,") {
			kind = spool
			continue
		}
		if strings.HasPrefix(record, "         ID") {
			continue
		}

		switch kind {
		case level1, level2:
			job := JobStatus{}
			err := job.parseLine(record, kind)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s record at line %d: %w", jesLevel(kind), i+1, err)
			}
			jobs = append(jobs, job)
		case spool:
			if isTargetJob && strings.Contains(record, ",") || strings.TrimSpace(record) == "" {
				continue
			}
			if !strings.HasPrefix(record, "    ") {
				cols := whitespaceRegex.Split(record, 2)
				if len(cols) < 1 {
					return nil, fmt.Errorf("invalid record: '%s' in line %d", record, i)
				}
				cnt, err := strconv.Atoi(cols[0])
				if err != nil {
					return nil, fmt.Errorf("error parsing record '%s': %s", record, err)
				}
				if cnt == 0 {
					return nil, fmt.Errorf("got zero spool count")
				}
				if len(spools) != cnt {
					return nil, fmt.Errorf("got %d spools, expected %d", len(spools), cnt)
				}
				continue
			}
			sp := JobSpool{}
			err := sp.parse(record)
			if err != nil {
				return nil, err
			}
			spools = append(spools, sp)
		}
	}
	if len(spools) > 0 {
		jobs[0].detail = spools
	}
	return jobs, nil
}

func (j *JobStatus) parseLine(line string, level int) error {
	fields := whitespaceRegex.Split(line, level)
	if len(fields) != level {
		return fmt.Errorf("invalid record: '%s'", line)
	}

	err := j.Name.parse(fields[0])
	if err != nil {
		return fmt.Errorf("failed to parse JobName field: %v", err)
	}

	err = j.JobId.parse(fields[1])
	if err != nil {
		return fmt.Errorf("failed to parse JobID field: %v", err)
	}

	if level == level1 {
		err = j.Status.parse(fields[2])
		if err != nil {
			return fmt.Errorf("failed to parse JobStatus field: %v", err)
		}

		err = j.Class.parse(fields[3])
		if err != nil {
			return fmt.Errorf("failed to parse JobClass field: %v", err)
		}

	} else {
		err = j.Owner.parse(fields[2])
		if err != nil {
			return fmt.Errorf("failed to parse JobOwner field: %v", err)
		}

		err = j.Status.parse(fields[3])
		if err != nil {
			return fmt.Errorf("failed to parse JobStatus field: %v", err)
		}

		err = j.Class.parse(fields[4])
		if err != nil {
			return fmt.Errorf("failed to parse JobClass field: %v", err)
		}
	}
	return nil
}

func (s *JobSpool) parse(line string) error {

	fields := whitespaceRegex.Split(strings.TrimSpace(line), 6)
	if len(fields) != 6 {
		return fmt.Errorf("invalid record: '%s'", line)
	}

	err := s.Id.parse(fields[0])
	if err != nil {
		return fmt.Errorf("failed to parse Id field: %v", err)
	}

	err = s.StepName.parse(fields[1])
	if err != nil {
		return fmt.Errorf("failed to parse StepName field: %v", err)
	}

	err = s.ProcSpec.parse(fields[2])
	if err != nil {
		return fmt.Errorf("failed to parse ProcSpec field: %v", err)
	}

	err = s.C.parse(fields[3])
	if err != nil {
		return fmt.Errorf("failed to parse C field: %v", err)
	}

	err = s.DDName.parse(fields[4])
	if err != nil {
		return fmt.Errorf("failed to parse DDName field: %v", err)
	}

	err = s.ByteCount.parse(fields[5])
	if err != nil {
		return fmt.Errorf("failed to parse ByteCount field: %v", err)
	}

	return nil
}

func jesLevel(level int) string {
	if level == level1 {
		return "JesInterfaceLevel=1"
	}
	return "JesInterfaceLevel=2"
}
