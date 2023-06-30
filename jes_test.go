package zftp_test

import (
	"regexp"
	"testing"

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
//STEP1   EXEC PGM=IEFBR14
//SYSOUT  DD SYSOUT=*
//*
//STEP2   EXEC PGM=IEBGENER
//SYSPRINT DD SYSOUT=*
//SYSIN   DD DUMMY
//SYSUT1  DD DUMMY
//SYSUT2  DD SYSOUT=*
//*
`

	// Submit the JCL
	output, err := s.SubmitJCL(jcl)
	if err != nil {
		t.Fatal(err)
	}
	jb := regexp.MustCompile(`JOB[0-9]+`)

	if !jb.MatchString(output) {
		t.Errorf("Empty output returned")
	}

	status, err := s.GetJobStatus(output)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Job status: %+v", status)
}
