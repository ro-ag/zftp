package zftp

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v0/hfs"
	"gopkg.in/ro-ag/zftp.v0/internal/utils"
	"io"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"
)

// JesOptions is an interface for options to the SubmitJCL method
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

// GetJobStatus retrieves the status of a JES job by ID.
func (s *FTPSession) GetJobStatus(jobID string) (*hfs.InfoJobDetail, error) {

	if utils.RegexSearchPattern.MatchString(jobID) {
		return nil, fmt.Errorf("invalid job-id: %s", jobID)
	}

	FileType, err := utils.SetValueAndGetCurrent("JES", s.SetStatusOf().FileType, s.StatusOf().FileType)
	if err != nil {
		return nil, err
	}
	defer FileType.Restore()

	JesJobName, err := utils.SetValueAndGetCurrent("*", s.SetStatusOf().JesJobName, s.StatusOf().JesJobName)
	if err != nil {
		return nil, err
	}
	defer JesJobName.Restore()

	records, err := s.List(jobID)
	if err != nil {
		return nil, err
	}

	return hfs.ParseInfoJobDetail(records)
}
