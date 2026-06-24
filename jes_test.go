// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"errors"
	"gopkg.in/ro-ag/zftp.v2"
	"gopkg.in/ro-ag/zftp.v2/hfs"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestSubmitJCL(t *testing.T) {
	requireEnv(t)
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
				if err != nil {
					t.Fatal(err)
				}
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
		n, err := s.RetrieveIO(job.ID, jobOutput, zftp.TypeAscii)

		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Retrieved %d bytes", n)

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

func TestSubmitLISTDS(t *testing.T) {
	requireEnv(t)
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
//LISTCAT JOB NOTIFY=&SYSUID,MSGLEVEL=(1,1)
//STEP1    EXEC PGM=IKJEFT01,DYNAMNBR=20
//SYSTSPRT DD  SYSOUT=*
//SYSTSIN  DD  *
  LISTDS 'Z33500.SAMPDATA.SEC.EBCDCIC'
/*
`
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
	t.Logf("Spool:\n%s", strings.Join(job.Spool, "\n")) // Print the spool

}

func TestSubmitDITTO(t *testing.T) {
	requireEnv(t)
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
//DITTOK JOB NOTIFY=&SYSUID,MSGLEVEL=(1,1)                                                          
//STEP1    EXEC PGM=DITTO,PARM='JOBSTREAM'                              
//STEPLIB  DD DSN=DIT.V1R3M0.SDITMOD1,DISP=SHR                     
//SYSPRINT DD SYSOUT=A                                             
//S10A     DD DUMMY                                                
//TAPEIN   DD DSN=CMDT.CASA.ARF.MILLION.TSOA.NAMES,VOL=SER=I02073, 
// UNIT=TAPE3480,DISP=SHR LABEL=(2,BLP,EXPDT=98000)                
//SYSIN    DD *                                                    
$$DITTO TLB INPUT=TAPEIN    
`
	job, err := s.SubmitJesGetByDSN(jcl)
	if !errors.Is(err, zftp.ErrIEF) {
		t.Errorf("Failed to submit JCL: %s", err)
	}

	if job.ReturnCode == 0 {
		t.Errorf("Expected job to failure, got ReturnCode %d", job.ReturnCode)
	}

	t.Logf("Job ID   : %s", job.ID)
	t.Logf("Job Name : %s", job.DisplayName)
	t.Logf("Job DSN  : %s", job.DSN)
	t.Logf("Job RC   : %d", job.ReturnCode)
	t.Logf("Expected Error: %s", err)
	t.Logf("Spool:\n%s", strings.Join(job.Spool, "\n")) // Print the spool
}

func TestPurgeJob_OK(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("DELE", "250 job JOB12345 purged")
	if err := s.PurgeJob("JOB12345"); err != nil {
		t.Fatalf("PurgeJob: %v", err)
	}
	cmds := srv.Commands()
	var idxSite, idxDele int = -1, -1
	for i, c := range cmds {
		if c == "SITE FILETYPE=JES" {
			idxSite = i
		}
		if c == "DELE JOB12345" {
			idxDele = i
		}
	}
	if idxSite < 0 {
		t.Fatalf("missing SITE FILETYPE=JES in %v", cmds)
	}
	if idxDele < 0 {
		t.Fatalf("missing DELE JOB12345 in %v", cmds)
	}
	if idxSite >= idxDele {
		t.Fatalf("SITE FILETYPE=JES must appear before DELE JOB12345; got %v", cmds)
	}
}

func TestPurgeJob_Error(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("DELE", "550 job not found")
	if err := s.PurgeJob("JOB12345"); !errors.Is(err, zftp.CodeError(550)) {
		t.Fatalf("PurgeJob err = %v, want CodeError(550)", err)
	}
}
