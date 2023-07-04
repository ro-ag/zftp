package zftp_test

import (
	"gopkg.in/ro-ag/zftp.v0/hfs"
	"regexp"
	"strings"
	"testing"
	"time"

	"gopkg.in/ro-ag/zftp.v0"
)

func TestSubmitJCL(t *testing.T) {
	// Create a new FTP session
	s, err := zftp.Open(hostname)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Login to the FTP server
	err = s.Login(username, password)
	if err != nil {
		t.Fatal(err)
	}

	// Define the JCL string
	jcl := `
//ANOTHER   JOB NOTIFY=&SYSUID,MSGLEVEL=(1,1)
//*
//STEP1    EXEC PGM=IEFBR14
//SYSOUT   DD SYSOUT=*
//*
//STEP2    EXEC PGM=IEBGENER
//SYSUT1   DD *
HELLO, WORLD!
/*
//SYSUT2   DD SYSOUT=*
//SYSIN    DD DUMMY
//SYSPRINT DD SYSOUT=*
//*
`

	t.Run("SubmitJCL and wait", func(t *testing.T) {
		// Submit the JCL

		job, err := s.SubmitJCL(jcl)
		if err != nil {
			t.Fatal(err)
		}
		jb := regexp.MustCompile(`JOB[0-9]+`)

		if !jb.MatchString(job.ID) {
			t.Errorf("Empty output returned")
		}

		// Wait for the job to complete

		status, err := s.GetJobStatus(job.ID)
		if err != nil {
			if err == hfs.ErrActiveJob {
				t.Logf("Job is still active")
				time.Sleep(5 * time.Second)
				status, err = s.GetJobStatus(job.ID)

			} else {
				t.Fatal(err)
			}
		}
		t.Logf("Job status: %+v", status)

		rc, err := status.ReturnCode()
		if err != nil {
			t.Fatal(err)
		}
		if rc != 0 {
			t.Errorf("Job failed with ReturnCode %d", rc)
		}

		// Get the job output

		s.SetStatusOf().FileType("JES")
		s.SetStatusOf().JesJobName("*")
		jobOutput := &strings.Builder{}
		n, str, err := s.RetrieveIO(job.ID, jobOutput, zftp.TypeAscii)

		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Retrieved %d bytes", n)
		t.Logf("Output: %s", str)

		t.Logf("Job output: %s", jobOutput.String())
	})
	// Get the job output by DSN

	t.Run("GetJobOutputByDSN", func(t *testing.T) {

		job, err := s.SubmitJesGetByDSN(jcl)
		if err != nil {
			t.Errorf("Failed to submit JCL: %s", err)
		}

		if job.ReturnCode != 0 {
			t.Errorf("Job failed with ReturnCode %d", job.ReturnCode)
		}

		t.Logf("Job ID   : %s", job.ID)
		t.Logf("Job Name : %s", job.DisplayName)
		t.Logf("Job DSN  : %s", job.DSN)
		t.Logf("Job RC   : %d", job.ReturnCode)

		for i := range job.Spool {
			t.Logf("Spool:\n%s", job.Spool[i])
		}

	})
}
