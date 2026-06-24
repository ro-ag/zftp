// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v2/hfs"
	"gopkg.in/ro-ag/zftp.v2/internal/utils"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// JesJob represents a JES job on the FTP server
type JesJob struct {
	// ID is the JES job id (e.g. JOB12345) parsed from the submit reply.
	ID string
	// DSN is the internal-reader dataset name the JCL was written to.
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

	curr, err := utils.SetValueAndGetCurrent(s.log, "JES", s.SetStatusOf().FileType, s.StatusOf().FileType)
	if err != nil {
		return nil, err
	}
	defer curr.Restore()

	_, msg, err := s.storeIO(job.DSN, jr, TypeAscii)
	if err != nil {
		return nil, fmt.Errorf("failed to write JCL to FTP server: %w", err)
	}

	match := s.jobPrefix.FindStringSubmatch(msg)
	if len(match) != 2 {
		return job, fmt.Errorf("failed to retrieve job-id from response: %s", msg)
	}

	job.ID = match[1]

	return job, nil
}

// SubmitJCL submits JCL to the FTP server and returns the Job-ID
func (s *FTPSession) SubmitJCL(jcl string, options ...JesSpec) (*JesJob, error) {
	return s.SubmitIO(strings.NewReader(jcl), options...)
}

// SubmitJCLFile submits JCL from a file to the FTP server and returns the Job-ID
// Optionally, JesSpec options to set additional parameters
func (s *FTPSession) SubmitJCLFile(jclFile string, options ...JesSpec) (*JesJob, error) {
	jcl, err := os.Open(jclFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read JCL file %s: %w", jclFile, err)
	}
	return s.SubmitIO(jcl, options...)
}

// JobResult is the outcome of a JES submit-and-fetch (SubmitJesGetByDSN): the
// submitted job plus its retrieved spool output, display name, and return code.
type JobResult struct {
	JesJob
	// Spool holds the job's spool files, one entry per spool dataset.
	Spool []string
	// DisplayName is the job name as reported in the spool output.
	DisplayName string
	// ReturnCode is the job's numeric return code, or -1 when the job did not end
	// with a parseable RC (e.g. an abend or a JCL/allocation error).
	ReturnCode int
}

var (
	jesJobDoneRegexp          = regexp.MustCompile(`When\s+(J\w+\d+)\s+is\s+done`)
	jesJobDoneRcRegexp        = regexp.MustCompile(`\$HASP395\s+(\w+)\s+ENDED\s+-\s+RC=(\d+)`)
	jesJobDoneEndedNoRcRegexp = regexp.MustCompile(`\$HASP395\s+(\w+)\s+ENDED`)
	// abendLineRegex matches a task-abend completion code (system Scde / user Ucde,
	// e.g. S0C4, S806, U0778) as written on IEF450I/IEF472I (and listing) lines:
	// "ABEND=Scde", "ABEND Scde", or "ABENDED Scde".
	abendLineRegex = regexp.MustCompile(`ABEND(?:ED)?[=\s]+[SU]?[0-9A-Fa-f]{3,4}`)
)

// SubmitJesGetByDSN puts JCL to the FTP server and returns the DSN, this uses StringToJCL internally
// it generates a unique job name and sets the site parameters to RECFM=FB LRECL=80 BLKSIZE=27920
// it returns the whole Spool output as a string
// this function waits for the job to complete
//
// NOTE: the RECFM/LRECL/BLKSIZE SITE attributes set here for the JCL upload are
// NOT restored afterwards (only FILETYPE and the JES job-name filter are), so they
// persist on the session for subsequent commands. Re-set them, or use a separate
// session, if a later transfer needs different allocation attributes.
func (s *FTPSession) SubmitJesGetByDSN(jcl string) (*JobResult, error) {
	currSeq, err := utils.SetValueAndGetCurrent(s.log, "SEQ", s.SetStatusOf().FileType, s.StatusOf().FileType)
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

	_, err = s.StoreIO(job.DSN, strings.NewReader(jcl), TypeAscii)
	if err != nil {
		return nil, fmt.Errorf("failed to write JCL to FTP server: %w", err)
	}

	currJes, err := utils.SetValueAndGetCurrent(s.log, "JES NOJESGETBYDSN", s.SetStatusOf().FileType, s.StatusOf().FileType)
	if err != nil {
		return nil, err
	}
	defer currJes.Restore()

	currName, err := utils.SetValueAndGetCurrent(s.log, "*", s.SetStatusOf().JesJobName, s.StatusOf().JesJobName)
	if err != nil {
		return nil, err
	}
	defer currName.Restore()

	jobOutput := &strings.Builder{}

	_, msg, err := s.retrieveIO(job.DSN, jobOutput, TypeAscii)
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
	errDetails, errType := classifyJesOutput(jobOutput.String())

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
	if err != nil {
		return job, fmt.Errorf("failed to parse job return code %q: %w", res[2], err)
	}
	return job, nil
}

// classifyJesOutput scans JES job output for a task abend (an alphanumeric Scde/
// Ucde completion code, usually on an IEF450I/IEF472I line), DFSMShsm Aggregate
// Backup (ABAxxx) messages, and allocation/JCL (IEFxxx) message identifiers. It
// returns the matched, trimmed lines together with the matching sentinel: ErrAbend
// takes precedence (an abend is the salient failure), otherwise ErrIEFAndABA when
// both IEF and ABA are present, else ErrIEF or ErrAba. It returns (nil, nil) when
// no such message is found. The sentinel is returned (not wrapped) so callers
// match it with errors.Is once it is wrapped with %w at the call site.
//
// Each line is attributed to a single category (abend first, then ABA, then IEF),
// so an IEF450I abend line is counted once and classified as an abend rather than
// as a generic IEF allocation message.
func classifyJesOutput(output string) (details []string, errType error) {
	lines := strings.Split(output, "\n")
	var sawAbend, sawIef, sawAba bool
	for _, line := range lines {
		matched := false
		if abendLineRegex.MatchString(line) {
			sawAbend, matched = true, true
		}
		if !matched {
			for key := range abaMessages {
				if strings.Contains(line, key) {
					sawAba, matched = true, true
					break
				}
			}
		}
		if !matched {
			for key := range iefMessages {
				if strings.Contains(line, key) {
					sawIef, matched = true, true
					break
				}
			}
		}
		if matched {
			details = append(details, strings.TrimSpace(line))
		}
	}
	switch {
	case sawAbend:
		errType = ErrAbend
	case sawIef && sawAba:
		errType = ErrIEFAndABA
	case sawIef:
		errType = ErrIEF
	case sawAba:
		errType = ErrAba
	}
	return details, errType
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
	FileType, err := utils.SetValueAndGetCurrent(s.log, "JES", s.SetStatusOf().FileType, s.StatusOf().FileType)
	if err != nil {
		return nil, err
	}
	defer FileType.Restore()

	// set jes job name to * and restore it after the function returns
	JesJobName, err := utils.SetValueAndGetCurrent(s.log, "*", s.SetStatusOf().JesJobName, s.StatusOf().JesJobName)
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

// WithJesEntryLimit sets the maximum number of entries to retrieve from JES
func WithJesEntryLimit(limit int) JesSpec {
	return jesOptionFunc(func(s *FTPSession) error {
		return s.SetStatusOf().JesEntryLimit(limit)
	})
}

// WithJesGetByDSN sets the JESGETBYDSN parameter to true or false
func WithJesGetByDSN(option bool) JesSpec {
	return jesOptionFunc(func(s *FTPSession) error {
		return s.SetStatusOf().JesGetByDSN(option)
	})
}

// WithJesLrecl sets the LRECL parameter for JES
func WithJesLrecl(len int) JesSpec {
	return jesOptionFunc(func(s *FTPSession) error {
		return s.SetStatusOf().JesLrecl(len)
	})
}

// WithJesPutGetTimeOut sets the timeout for JES PUT and GET commands
func WithJesPutGetTimeOut(seconds int) JesSpec {
	return jesOptionFunc(func(s *FTPSession) error {
		return s.SetStatusOf().JesPutGetTimeOut(seconds)
	})
}

// WithJesJobPattern changes the search pattern for the job-id in the response
// message. The default pattern is `(JOB\d{5})`. The pattern must contain exactly
// one capturing group — the job-id is read from submatch[1] — so a pattern with
// none or more than one group is rejected with an error rather than silently
// breaking job-id extraction.
func WithJesJobPattern(pattern string) JesSpec {
	return jesOptionFunc(func(s *FTPSession) error {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return err
		}
		if n := re.NumSubexp(); n != 1 {
			return fmt.Errorf("zftp: jes job pattern %q must have exactly one capturing group, has %d", pattern, n)
		}
		s.jobPrefix = re
		return nil
	})
}

// JesSpec is a JES submission option applied to a session before a job is
// submitted. Construct one with the With… helpers (e.g. WithJesEntryLimit,
// WithJesGetByDSN).
type JesSpec interface {
	Apply(*FTPSession) error
}

// jesOptionFunc adapts a plain function to the JesSpec interface. Callers use the
// With… constructors rather than this type directly.
type jesOptionFunc func(*FTPSession) error

func (f jesOptionFunc) Apply(s *FTPSession) error {
	return f(s)
}

// PurgeJob deletes a job from the JES spool by job id (DELE under FILETYPE=JES).
// The session's file type is set to JES for the call and restored afterward. A
// 550 (unknown job / not owner) is returned as a *ReturnError.
func (s *FTPSession) PurgeJob(jobID string) error {
	ft, err := utils.SetValueAndGetCurrent(s.log, "JES", s.SetStatusOf().FileType, s.StatusOf().FileType)
	if err != nil {
		return err
	}
	defer ft.Restore()
	_, err = s.SendCommand(CodeFileActionOK, "DELE", jobID)
	return err
}
