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

	// Submit the JCL
	jobID, err := s.SubmitJCL(jcl)
	if err != nil {
		t.Fatal(err)
	}
	jb := regexp.MustCompile(`JOB[0-9]+`)

	if !jb.MatchString(jobID) {
		t.Errorf("Empty output returned")
	}

	// Wait for the job to complete

	status, err := s.GetJobStatus(jobID)
	if err != nil {
		if err == hfs.ErrActiveJob {
			t.Logf("Job is still active")
			time.Sleep(5 * time.Second)
			status, err = s.GetJobStatus(jobID)

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
	n, str, err := s.RetrieveIO(jobID, jobOutput, zftp.TypeAscii)

	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Retrieved %d bytes", n)
	t.Logf("Output: %s", str)

	t.Logf("Job output: %s", jobOutput.String())

}
