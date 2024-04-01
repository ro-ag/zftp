package hfs

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	whitespaceRegex   = regexp.MustCompile(`\s+`)
	returnCodeRegex   = regexp.MustCompile(`RC=(\d+)`)
	abendCodeRegex    = regexp.MustCompile(`ABEND=(\d+)`)
	numberPrefixRegex = regexp.MustCompile(`^(\d+)\s+spool\s+files`)
)

const (
	jesInterfaceLevel1 = iota + 4
	jesInterfaceLevel2
)

var (
	ErrActiveJob  = errors.New("job is active")
	ErrAbendedJob = errors.New("job abended")
	ErrJCLError   = errors.New("job has JCL error")
)

// InfoJob represents a job record from the JES spool.
type InfoJob struct {
	Name   FieldString `json:"Name"`
	JobId  FieldString `json:"JobId"`
	Owner  FieldString `json:"Owner"`
	Status FieldString `json:"Status"`
	Class  FieldString `json:"Class"`
}

// String returns a row of text representing the job.
func (j *InfoJob) String() string {
	str := strings.Builder{}
	str.WriteString(fmt.Sprintf("Name: %s, ", j.Name.String()))
	str.WriteString(fmt.Sprintf("JobId: %s, ", j.JobId.String()))
	str.WriteString(fmt.Sprintf("Owner: %s, ", j.Owner.String()))
	str.WriteString(fmt.Sprintf("Status: %s, ", j.Status.String()))
	str.WriteString(fmt.Sprintf("Class: %s", j.Class.String()))
	return str.String()
}

// ParseInfoJob parses a slice of strings into a slice of InfoJob structs.
func ParseInfoJob(records []string) ([]InfoJob, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("no records provided")
	}

	kind := getInterfaceLevel(records[0])
	jobs := make([]InfoJob, 0)

	for i, record := range records {
		if i == 0 && kind == jesInterfaceLevel2 {
			continue
		}
		job, err := parseLineJob(record, kind)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s record at line %d: %w", jesLevel(kind), i+1, err)
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func parseLineJob(line string, level int) (j InfoJob, err error) {

	fields := whitespaceRegex.Split(line, level)
	if len(fields) != level {
		return j, fmt.Errorf("invalid record: '%s'", line)
	}

	err = j.Name.parse(fields[0])
	if err != nil {
		return j, fmt.Errorf("failed to parse JobName field: %v", err)
	}

	err = j.JobId.parse(fields[1])
	if err != nil {
		return j, fmt.Errorf("failed to parse JobID field: %v", err)
	}

	if level == jesInterfaceLevel1 {
		err = j.Status.parse(fields[2])
		if err != nil {
			return j, fmt.Errorf("failed to parse JobStatus field: %v", err)
		}

		err = j.Class.parse(fields[3])
		if err != nil {
			return j, fmt.Errorf("failed to parse JobClass field: %v", err)
		}

	} else {
		err = j.Owner.parse(fields[2])
		if err != nil {
			return j, fmt.Errorf("failed to parse JobOwner field: %v", err)
		}

		err = j.Status.parse(fields[3])
		if err != nil {
			return j, fmt.Errorf("failed to parse JobStatus field: %v", err)
		}

		err = j.Class.parse(fields[4])
		if err != nil {
			return j, fmt.Errorf("failed to parse JobClass field: %v", err)
		}
	}
	return j, nil
}

// InfoJobDetail contains the job record and the job details.
type InfoJobDetail struct {
	job    InfoJob
	detail []jobDetail
}

// Job returns the job record.
func (j InfoJobDetail) Job() InfoJob {
	return j.job
}

// Detail returns the job details.
func (j InfoJobDetail) Detail() []jobDetail {
	return j.detail
}

// ReturnCode returns the job return code.
// returns ErrActiveJob if the job is still active.
// returns ErrAbendedJob if the job abended.
func (j InfoJobDetail) ReturnCode() (rc int, err error) {

	if j.job.Status.String() == "ACTIVE" {
		return 0, ErrActiveJob
	}

	regex := returnCodeRegex

	if strings.Contains(j.job.Class.String(), "ABEND") {
		err = ErrAbendedJob
		regex = abendCodeRegex
	}

	if strings.Contains(j.job.Class.String(), "JCL error") {
		return -1, ErrJCLError
	}

	result := regex.FindStringSubmatch(j.job.Class.String())
	if len(result) != 2 {
		return 0, fmt.Errorf("no return code found")
	}

	rc, errInt := strconv.Atoi(result[1])
	if errInt != nil {
		return 0, fmt.Errorf("failed to parse return code: %w", errInt)
	}

	return rc, err
}

// InfoJobDetail returns the job details, contains the STEPS
type jobDetail struct {
	Id        FieldInt    `json:"Id"`
	StepName  FieldString `json:"StepName"`
	ProcSpec  FieldString `json:"ProcSpec"`
	C         FieldString `json:"C"`
	DDName    FieldString `json:"DDName"`
	ByteCount FieldInt    `json:"ByteCount"`
}

// String returns a row of text representing the job detail.
func (j jobDetail) String() string {
	str := strings.Builder{}
	str.WriteString(fmt.Sprintf("Id: %d, ", j.Id.Value()))
	str.WriteString(fmt.Sprintf("StepName: %s, ", j.StepName.String()))
	str.WriteString(fmt.Sprintf("ProcSpec: %s, ", j.ProcSpec.String()))
	str.WriteString(fmt.Sprintf("C: %s, ", j.C.String()))
	str.WriteString(fmt.Sprintf("DDName: %s, ", j.DDName.String()))
	str.WriteString(fmt.Sprintf("ByteCount: %d", j.ByteCount.Value()))
	return str.String()
}

// ParseInfoJobDetail parses a slice of strings into a slice of InfoJobDetail structs.
func ParseInfoJobDetail(records []string) (*InfoJobDetail, error) {
	if len(records) < 1 {
		return nil, fmt.Errorf("no records provided")
	}
	kind := getInterfaceLevel(records[0])
	if kind == jesInterfaceLevel1 {
		job, err := parseLineJob(records[0], kind)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s record: %w", jesLevel(kind), err)
		}
		return &InfoJobDetail{job: job}, nil
	}

	job, err := parseLineJob(records[1], kind)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s record: %w", jesLevel(kind), err)
	}

	jd := &InfoJobDetail{job: job}

	if jd.job.Status.String() == "ACTIVE" {
		return jd, ErrActiveJob
	}

	if len(records) < 3 {
		return jd, nil
	}

	if !strings.HasPrefix(records[2], "--------") {
		return jd, fmt.Errorf("cannot get spool detail: '%s'", records[2])
	}

	if len(records) < 4 {
		return jd, nil
	}

	if !strings.Contains(records[3], " ID  STEPNAME PROCSTEP") {
		return jd, fmt.Errorf("cannot get spool detail: '%s'", records[3])
	}

	if len(records) < 5 {
		return jd, nil
	}

	dt := make([]jobDetail, 0)

	for i := 4; i < len(records); i++ {
		if records[i] == "" {
			continue
		}

		if numberPrefixRegex.MatchString(records[i]) {
			spoolFiles := numberPrefixRegex.FindStringSubmatch(records[i])
			if len(spoolFiles) != 2 {
				return jd, fmt.Errorf("failed to parse %s record at line %d: %w", jesLevel(kind), i+1, err)
			}
			n, _ := strconv.Atoi(spoolFiles[1])
			if len(dt) != n {
				return jd, fmt.Errorf("the number of spool files is not equal to the number of records")
			}
			break
		}

		d, err := parseJobDetailLine(records[i])
		if err != nil {
			return jd, fmt.Errorf("failed to parse %s record at line %d: %w", jesLevel(kind), i+1, err)
		}
		dt = append(dt, d)
	}

	jd.detail = dt

	return jd, nil
}

func parseJobDetailLine(line string) (jobDetail, error) {
	s := jobDetail{}
	fields := whitespaceRegex.Split(strings.TrimSpace(line), 6)
	if len(fields) != 6 {
		return s, fmt.Errorf("invalid record: '%s'", line)
	}

	err := s.Id.parse(fields[0])
	if err != nil {
		return s, fmt.Errorf("failed to parse Id field: %v", err)
	}

	err = s.StepName.parse(fields[1])
	if err != nil {
		return s, fmt.Errorf("failed to parse StepName field: %v", err)
	}

	err = s.ProcSpec.parse(fields[2])
	if err != nil {
		return s, fmt.Errorf("failed to parse ProcSpec field: %v", err)
	}

	err = s.C.parse(fields[3])
	if err != nil {
		return s, fmt.Errorf("failed to parse C field: %v", err)
	}

	err = s.DDName.parse(fields[4])
	if err != nil {
		return s, fmt.Errorf("failed to parse DDName field: %v", err)
	}

	err = s.ByteCount.parse(fields[5])
	if err != nil {
		return s, fmt.Errorf("failed to parse ByteCount field: %v", err)
	}

	return s, nil
}

func jesLevel(level int) string {
	if level == jesInterfaceLevel1 {
		return "JesInterfaceLevel=1"
	}
	return "JesInterfaceLevel=2"
}

func getInterfaceLevel(line string) int {
	if strings.Contains(line, "JOBNAME") {
		return jesInterfaceLevel2
	}
	return jesInterfaceLevel1
}
