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

const (
	JesOptionHold      = true
	JesGenerateJobName = true
	JesOptionRelease   = true
)

func (s *FTPSession) submitJCL(jr io.Reader, options ...JesOptions) (string, error) {
	// Generate a unique job name
	jobName := generateJobName()

	// Submit the JCL using the generated job name

	_, err := s.Site("FILETYPE=JES", fmt.Sprintf("JESJOBNAME=%s", jobName))
	if err != nil {
		return "", fmt.Errorf("failed to submit JCL: %s", err)
	}

	// Write the JCL to the FTP server

	_, err = s.StoreIO("JES."+jobName, jr, TypeAscii)
	if err != nil {
		return "", fmt.Errorf("failed to write JCL to FTP server: %s", err)
	}

	// Retrieve the job output using JES quotes
	output, err := s.GetWithQuotes(jobName, 1)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve job output: %s", err)
	}

	return output, nil
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

	jobName = utils.StandardizeQuote(jobName)

	// Execute the JES quotes command
	output, err := s.Site("JESGET QUOTE", jobName, fmt.Sprint(maxLines))
	if err != nil {
		return "", fmt.Errorf("failed to execute JESGET command: %s", err)
	}

	// Check if the command was successful
	if !jesRegexOutput.MatchString(output) {
		return "", fmt.Errorf("failed to retrieve job output using JES quotes: %s", output)
	}

	// Use regular expressions to extract the job output
	output = jesRegexOutput.FindStringSubmatch(output)[1]

	return output, nil
}

// Generate a unique job name based on timestamp and random number
func generateJobName() string {
	timestamp := time.Now().Format("20060102150405") // Format: YYYYMMDDHHMMSS
	randomNumber := rand.Intn(99)                    // Generate a random number between 0 and 999
	jobName := fmt.Sprintf("JOB%.3s%.02d", timestamp, randomNumber)
	return jobName
}
