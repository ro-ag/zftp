// SPDX-License-Identifier: Apache-2.0

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

// Sentinels reported by InfoJobDetail.ReturnCode (and ParseInfoJobDetail) for a
// job that has not completed normally. They are returned directly, so callers
// match them with errors.Is, e.g. errors.Is(err, hfs.ErrAbendedJob).
var (
	// ErrActiveJob indicates the job is still running, so no return code is
	// available yet.
	ErrActiveJob = errors.New("job is active")
	// ErrAbendedJob indicates the job abended (its class reports an ABEND).
	ErrAbendedJob = errors.New("job abended")
	// ErrJCLError indicates the job failed with a JCL error.
	ErrJCLError = errors.New("job has JCL error")
)

// InfoJob represents a job record from the JES spool.
//
// SpoolFiles holds the spool-file count reported by a JesInterfaceLevel=1 listing,
// whose trailing column is "N spool files" rather than a job class. For such
// records Class is empty and SpoolFiles is N; for JesInterfaceLevel=2 records,
// which carry a real class, SpoolFiles is 0.
type InfoJob struct {
	// Name is the job name (the JOBNAME column).
	Name FieldString `json:"Name"`
	// JobId is the JES job identifier (e.g. "JOB12345").
	JobId FieldString `json:"JobId"`
	// Owner is the user ID that owns the job. It is empty for
	// JesInterfaceLevel=1 listings, which omit the owner column.
	Owner FieldString `json:"Owner"`
	// Status is the job status (e.g. "ACTIVE", "OUTPUT").
	Status FieldString `json:"Status"`
	// Class is the job class for JesInterfaceLevel=2 listings; it is empty for
	// JesInterfaceLevel=1 listings, which report a spool-file count instead.
	Class FieldString `json:"Class"`
	// SpoolFiles is the spool-file count from a JesInterfaceLevel=1 listing; it
	// is 0 for JesInterfaceLevel=2 listings.
	SpoolFiles int `json:"SpoolFiles,omitempty"`
}

// InfoJob and JobDetail satisfy fmt.Stringer by value (ListSpool returns
// []InfoJob; InfoJobDetail.Detail returns []JobDetail).
var (
	_ fmt.Stringer = InfoJob{}
	_ fmt.Stringer = JobDetail{}
)

// String returns a row of text representing the job.
func (j InfoJob) String() string {
	str := strings.Builder{}
	str.WriteString(fmt.Sprintf("Name: %s, ", j.Name.String()))
	str.WriteString(fmt.Sprintf("JobId: %s, ", j.JobId.String()))
	str.WriteString(fmt.Sprintf("Owner: %s, ", j.Owner.String()))
	str.WriteString(fmt.Sprintf("Status: %s, ", j.Status.String()))
	str.WriteString(fmt.Sprintf("Class: %s", j.Class.String()))
	if j.SpoolFiles > 0 {
		str.WriteString(fmt.Sprintf(", SpoolFiles: %d", j.SpoolFiles))
	}
	return str.String()
}

// ParseInfoJob parses a slice of strings into a slice of InfoJob structs.
// Blank lines (common when the raw server response is split on newlines, e.g. a
// trailing newline) are ignored so callers need not pre-trim the input.
func ParseInfoJob(records []string) ([]InfoJob, error) {
	// Keep each retained line paired with its 1-based position in the ORIGINAL
	// input, so a parse error reports the real line number rather than the index
	// after blank lines were filtered out.
	type lineAt struct {
		text string
		orig int
	}
	cleaned := make([]lineAt, 0, len(records))
	for i, r := range records {
		if strings.TrimSpace(r) != "" {
			cleaned = append(cleaned, lineAt{text: r, orig: i + 1})
		}
	}
	if len(cleaned) == 0 {
		return nil, fmt.Errorf("no records provided")
	}

	kind := getInterfaceLevel(cleaned[0].text)
	jobs := make([]InfoJob, 0, len(cleaned))

	for i, record := range cleaned {
		if i == 0 && kind == jesInterfaceLevel2 {
			continue
		}
		job, err := parseLineJob(record.text, kind)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s record at line %d: %w", jesLevel(kind), record.orig, err)
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
		return j, fmt.Errorf("failed to parse JobName field: %w", err)
	}

	err = j.JobId.parse(fields[1])
	if err != nil {
		return j, fmt.Errorf("failed to parse JobID field: %w", err)
	}

	if level == jesInterfaceLevel1 {
		err = j.Status.parse(fields[2])
		if err != nil {
			return j, fmt.Errorf("failed to parse JobStatus field: %w", err)
		}

		// A JesInterfaceLevel=1 record has no Class column; its trailing column is
		// the spool-file count ("N spool files"). Parse the count into SpoolFiles
		// and leave Class empty.
		m := numberPrefixRegex.FindStringSubmatch(strings.TrimSpace(fields[3]))
		if m == nil {
			return j, fmt.Errorf("failed to parse spool-file count: %q", fields[3])
		}
		j.SpoolFiles, err = strconv.Atoi(m[1])
		if err != nil {
			return j, fmt.Errorf("failed to parse spool-file count %q: %w", m[1], err)
		}

	} else {
		err = j.Owner.parse(fields[2])
		if err != nil {
			return j, fmt.Errorf("failed to parse JobOwner field: %w", err)
		}

		err = j.Status.parse(fields[3])
		if err != nil {
			return j, fmt.Errorf("failed to parse JobStatus field: %w", err)
		}

		err = j.Class.parse(fields[4])
		if err != nil {
			return j, fmt.Errorf("failed to parse JobClass field: %w", err)
		}
	}
	return j, nil
}

// InfoJobDetail contains the job record and the job details.
type InfoJobDetail struct {
	job    InfoJob
	detail []JobDetail
}

// Job returns the job record.
func (j InfoJobDetail) Job() InfoJob {
	return j.job
}

// Detail returns the job details.
func (j InfoJobDetail) Detail() []JobDetail {
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
		// An ABEND with no parseable numeric code (e.g. "ABEND S0C4") must still
		// report ErrAbendedJob rather than a generic "no return code found".
		if err != nil {
			return -1, err
		}
		return 0, fmt.Errorf("no return code found")
	}

	rc, errInt := strconv.Atoi(result[1])
	if errInt != nil {
		return 0, fmt.Errorf("failed to parse return code: %w", errInt)
	}

	return rc, err
}

// JobDetail represents one step's spool detail within a job (the columns of a
// JesInterfaceLevel=2 job listing): step id, step name, procedure step, class,
// DD name, and byte count.
type JobDetail struct {
	// Id is the spool dataset's sequence number within the job (the ID column).
	Id FieldInt `json:"Id"`
	// StepName is the job step name (the STEPNAME column).
	StepName FieldString `json:"StepName"`
	// ProcSpec is the procedure step name (the PROCSTEP column).
	ProcSpec FieldString `json:"ProcSpec"`
	// C is the SYSOUT class of the spool dataset (the single-character C column).
	C FieldString `json:"C"`
	// DDName is the DD name of the spool dataset (the DDNAME column).
	DDName FieldString `json:"DDName"`
	// ByteCount is the size of the spool dataset, in bytes (the BYTE-COUNT
	// column).
	ByteCount FieldInt `json:"ByteCount"`
}

// String returns a row of text representing the job detail.
func (j JobDetail) String() string {
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

	// Level 2: records[0] is the column header. Walk the remaining records by
	// content — skipping blank lines — rather than by fixed offsets, so a stray
	// blank line does not break parsing. i tracks the position in records, so
	// detail parse errors still report a 1-based line number (i+1).
	i := 1
	skipBlank := func() {
		for i < len(records) && strings.TrimSpace(records[i]) == "" {
			i++
		}
	}

	skipBlank()
	if i >= len(records) {
		return nil, fmt.Errorf("no %s job record found", jesLevel(kind))
	}
	job, err := parseLineJob(records[i], kind)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s record: %w", jesLevel(kind), err)
	}
	i++

	jd := &InfoJobDetail{job: job}

	if jd.job.Status.String() == "ACTIVE" {
		return jd, ErrActiveJob
	}

	// A "--------" separator introduces the detail section; its absence simply
	// means there is no detail to parse.
	skipBlank()
	if i >= len(records) {
		return jd, nil
	}
	if !strings.HasPrefix(records[i], "--------") {
		return jd, fmt.Errorf("cannot get spool detail: '%s'", records[i])
	}
	i++

	// The detail column header follows the separator.
	skipBlank()
	if i >= len(records) {
		return jd, nil
	}
	if !strings.Contains(records[i], " ID  STEPNAME PROCSTEP") {
		return jd, fmt.Errorf("cannot get spool detail: '%s'", records[i])
	}
	i++

	dt := make([]JobDetail, 0)
	for ; i < len(records); i++ {
		if strings.TrimSpace(records[i]) == "" {
			continue
		}

		if m := numberPrefixRegex.FindStringSubmatch(strings.TrimSpace(records[i])); m != nil {
			n, _ := strconv.Atoi(m[1])
			if len(dt) != n {
				return jd, fmt.Errorf("spool-file count (%d) does not match the number of detail records (%d)", n, len(dt))
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

func parseJobDetailLine(line string) (JobDetail, error) {
	s := JobDetail{}
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
