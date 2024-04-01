package zftp

import (
	"errors"
	"fmt"
	"gopkg.in/ro-ag/zftp.v0/hfs"
	"gopkg.in/ro-ag/zftp.v0/internal/utils"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var jesJobNameRegexp = regexp.MustCompile(`(JOB\d{5})`)

// JesJob represents a JES job on the FTP server
type JesJob struct {
	ID  string
	DSN string
}

// SubmitIO submits JCL using a reader to the FTP server and returns the Job-ID
// returns the Job-ID and the response message
func (s *FTPSession) SubmitIO(jr io.Reader, options ...JesSpec) (*JesJob, error) {
	// Generate a unique job name
	job := &JesJob{}
	job.DSN = generateJobFileName()

	for _, opt := range options {
		err := opt.Apply(s)
		if err != nil {
			return nil, err
		}
	}

	curr, err := utils.SetValueAndGetCurrent("JES", s.SetStatusOf().FileType, s.StatusOf().FileType)
	if err != nil {
		return nil, err
	}
	defer curr.Restore()

	_, msg, err := s.StoreIO(job.DSN, jr, TypeAscii)
	if err != nil {
		return nil, fmt.Errorf("failed to write JCL to FTP server: %w", err)
	}

	match := jesJobNameRegexp.FindStringSubmatch(msg)
	if len(match) != 2 {
		return job, fmt.Errorf("failed to retrieve job-id from response: %s", msg)
	}

	job.ID = match[1]

	return job, nil
}

// SubmitJCL submits JCL to the FTP server and returns the Job-ID
func (s *FTPSession) SubmitJCL(jcl string) (*JesJob, error) {
	return s.SubmitIO(strings.NewReader(jcl))
}

func (s *FTPSession) SubmitJCLFile(jclFile string) (*JesJob, error) {
	jcl, err := os.Open(jclFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read JCL file %s: %w", jclFile, err)
	}
	return s.SubmitIO(jcl)
}

type JobResult struct {
	JesJob
	Spool       []string
	DisplayName string
	ReturnCode  int
}

var (
	jesJobDoneRegexp          = regexp.MustCompile(`When\s+(J\w+\d+)\s+is\s+done`)
	jesJobDoneRcRegexp        = regexp.MustCompile(`\$HASP395\s+(\w+)\s+ENDED\s+-\s+RC=(\d+)`)
	jesJobDoneEndedNoRcRegexp = regexp.MustCompile(`\$HASP395\s+(\w+)\s+ENDED`)
)

// SubmitJesGetByDSN puts JCL to the FTP server and returns the DSN, this uses StringToJCL internally
// it generates a unique job name and sets the site parameters to RECFM=FB LRECL=80 BLKSIZE=27920
// it returns the whole Spool output as a string
// this function waits for the job to complete
func (s *FTPSession) SubmitJesGetByDSN(jcl string) (*JobResult, error) {
	currSeq, err := utils.SetValueAndGetCurrent("SEQ", s.SetStatusOf().FileType, s.StatusOf().FileType)
	if err != nil {
		return nil, err
	}
	defer currSeq.Restore()

	_, err = s.Site("RECFM=FB LRECL=80 BLKSIZE=27920")
	if err != nil {
		return nil, fmt.Errorf("failed to set site parameters: %w", err)
	}

	job := &JobResult{}

	job.DSN = generateJobFileName()

	_, _, err = s.StoreIO(job.DSN, strings.NewReader(jcl), TypeAscii)
	if err != nil {
		return nil, fmt.Errorf("failed to write JCL to FTP server: %w", err)
	}

	currJes, err := utils.SetValueAndGetCurrent("JES NOJESGETBYDSN", s.SetStatusOf().FileType, s.StatusOf().FileType)
	if err != nil {
		return nil, err
	}
	defer currJes.Restore()

	currName, err := utils.SetValueAndGetCurrent("*", s.SetStatusOf().JesJobName, s.StatusOf().JesJobName)
	if err != nil {
		return nil, err
	}
	defer currName.Restore()

	jobOutput := &strings.Builder{}

	_, msg, err := s.RetrieveIO(job.DSN, jobOutput, TypeAscii)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve job output: %w", err)
	}

	/* get job-id from response */
	res := jesJobDoneRegexp.FindStringSubmatch(msg)
	if len(res) != 2 {
		return nil, fmt.Errorf("failed to retrieve job-id from response: %s", msg)
	}

	job.ID = res[1]

	spool := strings.TrimSpace(jobOutput.String())
	spool = strings.TrimSuffix(spool, " !! END OF JES SPOOL FILE !!")
	job.Spool = strings.Split(spool, " !! END OF JES SPOOL FILE !!")
	for i := range job.Spool {
		job.Spool[i] = strings.TrimSpace(job.Spool[i])
	}

	/* check if job has ended */
	if !jesJobDoneEndedNoRcRegexp.MatchString(jobOutput.String()) {
		return job, fmt.Errorf("job has not ended: %s", msg)
	}

	/* get job name from response */
	res = jesJobDoneEndedNoRcRegexp.FindStringSubmatch(jobOutput.String())
	job.DisplayName = res[1]

	/* analyze for job abending or IEF errors */

	errDetails := make([]string, 0)
	var errType error
	lines := strings.Split(jobOutput.String(), "\n")
	/* analyze if job has abend errors */
	for _, line := range lines {
		for key := range abaMessages {
			if strings.Contains(line, key) {
				errDetails = append(errDetails, strings.TrimSpace(line))
				errType = ErrAba
			}
		}
	}

	/* analyze if job has errors */
	for _, line := range lines {
		for key := range iefMessages {
			if strings.Contains(line, key) {
				errDetails = append(errDetails, strings.TrimSpace(line))
				if errors.Is(errType, ErrAba) || errors.Is(errType, ErrIEFAndABA) {
					errType = ErrIEFAndABA
				} else {
					errType = ErrIEF
				}
			}
		}
	}

	/* get job return code from response */
	res = jesJobDoneRcRegexp.FindStringSubmatch(jobOutput.String())
	if len(res) != 3 {
		job.ReturnCode = -1
		if errType != nil {
			return job, fmt.Errorf("%s: %w", strings.Join(errDetails, ": "), errType)
		}
		return job, fmt.Errorf("failed to retrieve job-id from response: %s", msg)
	}

	job.ReturnCode, err = strconv.Atoi(res[2])
	return job, nil
}

// Generate a unique job name based on timestamp and random number
func generateJobFileName() string {
	currentTime := time.Now()

	// Format timestamp to YYMMDD and HHMMSS formats
	date := currentTime.Format("060102")
	hour := currentTime.Format("150405")

	// Use nanoseconds for unique identifier, limit to 7 digits by dividing by 10
	uniqueID := currentTime.Nanosecond() / 10

	// Combine date, time and uniqueID to form jobName
	ID := fmt.Sprintf("%.07d", uniqueID)
	jobName := fmt.Sprintf("JES.D%.6s.T%.6s.N%.7s", date, hour, ID[:7])

	return jobName
}

// GetJobStatus retrieves the status of a JES job by ID.
// This function is setting the LIST parameters to retrieve JOBS instead of FILES.
//
// It restores the original "global" list parameters after the function returns.
func (s *FTPSession) GetJobStatus(jobID string) (*hfs.InfoJobDetail, error) {

	// validate the job-id format is correct
	if utils.RegexSearchPattern.MatchString(jobID) {
		return nil, fmt.Errorf("invalid job-id: %s", jobID)
	}

	// set JES parameters and restore them after the function returns
	FileType, err := utils.SetValueAndGetCurrent("JES", s.SetStatusOf().FileType, s.StatusOf().FileType)
	if err != nil {
		return nil, err
	}
	defer FileType.Restore()

	// set jes job name to * and restore it after the function returns
	JesJobName, err := utils.SetValueAndGetCurrent("*", s.SetStatusOf().JesJobName, s.StatusOf().JesJobName)
	if err != nil {
		return nil, err
	}
	defer JesJobName.Restore()

	// list for job details
	records, err := s.List(jobID)
	if err != nil {
		return nil, err
	}

	jr, err := hfs.ParseInfoJobDetail(records)
	return jr, err
}

func WithJesEntryLimit(limit int) JesSpec {
	return JesOptionFunc(func(s *FTPSession) error {
		return s.SetStatusOf().JesEntryLimit(limit)
	})
}

func WithJesGetByDSN(option bool) JesSpec {
	return JesOptionFunc(func(s *FTPSession) error {
		return s.SetStatusOf().JesGetByDSN(option)
	})
}

func WithJesLrecl(len int) JesSpec {
	return JesOptionFunc(func(s *FTPSession) error {
		return s.SetStatusOf().JesLrecl(len)
	})
}

func WithJesPutGetTimeOut(seconds int) JesSpec {
	return JesOptionFunc(func(s *FTPSession) error {
		return s.SetStatusOf().JesPutGetTimeOut(seconds)
	})
}

type JesSpec interface {
	Apply(*FTPSession) error
}

type JesOptionFunc func(*FTPSession) error

func (f JesOptionFunc) Apply(s *FTPSession) error {
	return f(s)
}
