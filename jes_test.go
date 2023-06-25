package zftp_test

import (
	"fmt"
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
	msg := ""
	/*
		msg, _ = s.SendCommand(zftp.CodeHelpMsg, "HELP")
		t.Log(msg)
		msg, _ = s.SendCommand(zftp.CodeHelpMsg, "HELP SITE")
		t.Log(msg)

	*/
	msg, err = s.Stat()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(msg)

	//s.SendCommand(zftp.CodeCmdOK, "SITE FILETYPE=JES")
	if ft, err := s.StatusOf().FILEtype(); err != nil {
		t.Fatal(err)
	} else {
		t.Log(ft)
	}

	s.Site("FILETYPE=JES")
	//s.SendCommand(zftp.CodeCmdOK, "SITE QUOTE Q")
	s.Site("JESJOBNAME=*")
	s.Site("JESOWNER=*")
	s.Site("JESSTATUS=ALL")
	//s.Site("HELP DIR")
	s.SetType(zftp.TypeAscii)
	s.List("*")
	return

	// Define the JCL string
	jcl := `
//HELLO   JOB NOTIFY=&SYSUID,MSGLEVEL=(1,1)
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

	// Check if the output contains the expected message
	expectedOutput := "JOB HELLO (JOB12345) SUBMITTED"
	if !containsSubstring(output, expectedOutput) {
		t.Errorf("Unexpected output. Expected: %s, Got: %s", expectedOutput, output)
	}

	regexp.MustCompile(`JOB[0-9]+`)
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}
