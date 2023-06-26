package zftp

import (
	"fmt"
	log "github.com/sirupsen/logrus"
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

const (
	JesOptionHold      = true
	JesGenerateJobName = true
	JesOptionRelease   = true
)

var jesJobNameRegexp = regexp.MustCompile(`(JOB\d{5})`)

func (s *FTPSession) submitJCL(jr io.Reader, options ...JesOptions) (string, error) {
	// Generate a unique job name
	jobFileName := generateJobFileName()

	currFiletype, err := s.StatusOf().FILEtype()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve current FILETYPE: %s", err)
	}
	defer s.setFleTypeORLog(currFiletype)

	// Submit the JCL using the generated job name
	_, err = s.Site("FILETYPE=JES")
	if err != nil {
		return "", fmt.Errorf("failed to submit JCL: %s", err)
	}

	_, msg, err := s.StoreIO("JES."+jobFileName, jr, TypeAscii)
	if err != nil {
		return "", fmt.Errorf("failed to write JCL to FTP server: %s", err)
	}

	match := jesJobNameRegexp.FindStringSubmatch(msg)
	if len(match) != 2 {
		return "", fmt.Errorf("failed to retrieve job name from response: %s", msg)
	}

	return match[1], nil
}

func (s *FTPSession) SubmitJCL(jcl string, options ...JesOptions) (string, error) {
	return s.submitJCL(strings.NewReader(jcl))
}

func (s *FTPSession) PutSubmitJCL(jclFile string, options ...JesOptions) (string, error) {
	jcl, err := os.Open(jclFile)
	if err != nil {
		return "", fmt.Errorf("failed to read JCL file: %s", err)
	}
	return s.submitJCL(jcl)
}

var (
	jesRegexOutput = regexp.MustCompile(`(?m)^OUTPUT STARTS[\r\n]+(.*)[\r\n]+OUTPUT ENDS$`)
)

// GetWithQuotes retrieves the output of a JCL job using JES quotes
func (s *FTPSession) GetWithQuotes(jobName string, maxLines int) (string, error) {
	// Create the JES quotes command

	//jobName = utils.StandardizeQuote(jobName)
	output := strings.Builder{}

	jn := fmt.Sprintf("'%s.OUTPUT'", jobName)

	s.RetrieveIO(jn, &output, TypeAscii)

	// Check if the command was successful
	if !jesRegexOutput.MatchString(output.String()) {
		return "", fmt.Errorf("failed to retrieve job output using JES quotes: %s", output.String())
	}

	// Use regular expressions to extract the job output
	// output = jesRegexOutput.FindStringSubmatch(output.String())[1]

	return "", nil
}

// Generate a unique job name based on timestamp and random number
func generateJobFileName() string {
	timestamp := time.Now().Format("20060102150405") // Format: YYYYMMDDHHMMSS
	randomNumber := rand.Intn(99)                    // Generate a random number between 0 and 999
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
	// Send command to list the jobs
	resp, err := s.SendCommand(CodeCmdOK, "LIST", jobName)
	if err != nil {
		return nil, fmt.Errorf("failed to send LIST command: %w", err)
	}

	// Parse the response to extract job status
	lines := strings.Split(resp, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == jobName {
			return &JobStatus{Name: fields[0], Owner: fields[1], Status: fields[2]}, nil
		}
	}

	return nil, fmt.Errorf("failed to find job status for %s", jobName)
}

func (s *FTPSession) setFleTypeORLog(currFiletype string) {
	_, err := s.Site("FILETYPE=" + currFiletype)
	if err != nil {
		log.Errorf("failed to restore FILETYPE: %s", err)
	}
}
