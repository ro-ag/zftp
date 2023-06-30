package zftp

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v0/internal/utils"
	"io"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"
)

type JesOptions interface {
	option() bool
}

var jesJobNameRegexp = regexp.MustCompile(`(JOB\d{5})`)

// submitJcl submits JCL to the FTP server and returns the Job-ID
func (s *FTPSession) submitJcl(jr io.Reader, options ...JesOptions) (string, error) {
	// Generate a unique job name
	jobFileName := generateJobFileName()
	curr, err := utils.SetValueAndGetCurrent("JES", s.SetStatusOf().FileType, s.StatusOf().FileType)
	if err != nil {
		return "", err
	}
	defer curr.Restore()

	_, msg, err := s.StoreIO(jobFileName, jr, TypeAscii)
	if err != nil {
		return "", fmt.Errorf("failed to write JCL to FTP server: %s", err)
	}

	match := jesJobNameRegexp.FindStringSubmatch(msg)
	if len(match) != 2 {
		return "", fmt.Errorf("failed to retrieve job-id from response: %s", msg)
	}

	return match[1], nil
}

// SubmitJCL submits JCL to the FTP server and returns the Job-ID
func (s *FTPSession) SubmitJCL(jcl string, options ...JesOptions) (string, error) {
	return s.submitJcl(strings.NewReader(jcl))
}

func (s *FTPSession) SubmitJCLFile(jclFile string, options ...JesOptions) (string, error) {
	jcl, err := os.Open(jclFile)
	if err != nil {
		return "", fmt.Errorf("failed to read JCL file: %s", err)
	}
	return s.submitJcl(jcl)
}

// Generate a unique job name based on timestamp and random number
func generateJobFileName() string {
	timestamp := time.Now().Format("20060102150405") // Format: YYYYMMDDHHMMSS
	randomNumber := rand.Intn(999999)                // Generate a random number between 0 and 999
	jobName := fmt.Sprintf("JES.D%.7s.N#%.06d", timestamp, randomNumber)
	return jobName
}

// JobStatus holds the status of a job.
type JobStatus struct {
	Name   string
	Status string
	Owner  string
	// Include other fields as needed
}

// GetJobStatus retrieves the status of a JES job.
func (s *FTPSession) GetJobStatus(jobName string) (*JobStatus, error) {

	curr, err := utils.SetValueAndGetCurrent("*", s.SetStatusOf().JesJobName, s.StatusOf().JesJobName)
	if err != nil {
		return nil, err
	}
	defer curr.Restore()

	p, err := s.ListSpool(jobName)
	if err != nil {
		return nil, err
	}

	if len(p) == 0 {
		return nil, fmt.Errorf("failed to find job status for %s", jobName)
	}
	if len(p) > 1 {
		return nil, fmt.Errorf("unexpected return from spool list %s", jobName)
	}

	if len(p) == 1 {
		return &JobStatus{
			Name:   jobName,
			Status: p[0].Status.Value(),
			Owner:  p[0].Owner.Value(),
		}, nil
	}
	return nil, fmt.Errorf("failed to find job status for %s", jobName)
}
