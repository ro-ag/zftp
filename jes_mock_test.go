// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"errors"
	"strings"
	"testing"

	zftp "gopkg.in/ro-ag/zftp.v2"
)

// TestSubmitJCL_ExtractsJobID submits JCL over the internal reader (STOR in JES
// filetype) and verifies the job id is pulled from the z/OS submit reply
// ("IT IS KNOWN TO JES AS JOBnnnnn") by the default (JOB\d{5}) pattern.
func TestSubmitJCL_ExtractsJobID(t *testing.T) {
	s, srv := dialMock(t)
	srv.CompletionReply("STOR",
		"250-IT IS KNOWN TO JES AS JOB12345",
		"250 SUBMIT successful, job submitted")

	job, err := s.SubmitJCL("//RUN JOB (ACCT),CLASS=A\n//S1 EXEC PGM=IEFBR14\n")
	if err != nil {
		t.Fatalf("SubmitJCL: %v", err)
	}
	if job.ID != "JOB12345" {
		t.Errorf("job.ID = %q, want JOB12345", job.ID)
	}
	// The submit must go out in JES filetype (SITE FILETYPE=JES) before the STOR.
	if !hasCmd(srv.Commands(), "SITE FILETYPE=JES") {
		t.Errorf("expected SITE FILETYPE=JES before submit; commands=%v", srv.Commands())
	}
}

// TestGetJobStatus_Level2Detail lists a single job's status (JesInterfaceLevel=2)
// and verifies the record and its return code are parsed: GetJobStatus must set
// JES filetype, list the job id, and hand the listing to ParseInfoJobDetail.
func TestGetJobStatus_Level2Detail(t *testing.T) {
	s, srv := dialMock(t)
	detail := "JOBNAME  JOBID    OWNER    STATUS   CLASS\r\n" +
		"MYJOB    JOB12345 ME       OUTPUT   RC=0000\r\n"
	srv.DataFor("LIST", "JOB12345", detail)

	jd, err := s.GetJobStatus("JOB12345")
	if err != nil {
		t.Fatalf("GetJobStatus: %v", err)
	}
	jb := jd.Job()
	if got := jb.JobId.String(); got != "JOB12345" {
		t.Errorf("JobId = %q, want JOB12345", got)
	}
	if got := jb.Name.String(); got != "MYJOB" {
		t.Errorf("Name = %q, want MYJOB", got)
	}
	rc, err := jd.ReturnCode()
	if err != nil {
		t.Fatalf("ReturnCode: %v", err)
	}
	if rc != 0 {
		t.Errorf("ReturnCode = %d, want 0", rc)
	}
}

// TestSubmitJesGetByDSN_Success parses a clean job: the job id comes from the
// control reply, the display name and RC=0000 from the spool payload.
func TestSubmitJesGetByDSN_Success(t *testing.T) {
	s, srv := dialMock(t)
	srv.CompletionReply("RETR",
		"250-It is known to JES as JOB12345",
		"250-When JOB12345 is done, retrieval of its output begins",
		"250 transfer complete")
	spool := "1                    J E S 2  J O B  L O G\n" +
		"$HASP395 MYJOB ENDED - RC=0000\n" +
		" !! END OF JES SPOOL FILE !!\n"
	srv.DataFor("RETR", "", spool)

	job, err := s.SubmitJesGetByDSN("//MYJOB JOB (ACCT)\n//S1 EXEC PGM=IEFBR14\n")
	if err != nil {
		t.Fatalf("SubmitJesGetByDSN: %v", err)
	}
	if job.ID != "JOB12345" {
		t.Errorf("job.ID = %q, want JOB12345", job.ID)
	}
	if job.DisplayName != "MYJOB" {
		t.Errorf("job.DisplayName = %q, want MYJOB", job.DisplayName)
	}
	if job.ReturnCode != 0 {
		t.Errorf("job.ReturnCode = %d, want 0", job.ReturnCode)
	}
}

// TestSubmitJesGetByDSN_IEFError classifies an allocation/JCL (IEFxxx) failure:
// the spool carries an IEF message and a HASP395 ENDED line with no parseable RC,
// so the result reports ErrIEF and ReturnCode -1.
func TestSubmitJesGetByDSN_IEFError(t *testing.T) {
	s, srv := dialMock(t)
	srv.CompletionReply("RETR",
		"250-When JOB12345 is done, retrieval of its output begins",
		"250 transfer complete")
	spool := "IEF001I JOB FAILED - ALLOCATION ERROR\n" +
		"$HASP395 MYJOB ENDED\n" +
		" !! END OF JES SPOOL FILE !!\n"
	srv.DataFor("RETR", "", spool)

	job, err := s.SubmitJesGetByDSN("//MYJOB JOB (ACCT)\n//S1 EXEC PGM=IEFBR14\n")
	if !errors.Is(err, zftp.ErrIEF) {
		t.Fatalf("err = %v, want ErrIEF", err)
	}
	if job.ReturnCode != -1 {
		t.Errorf("job.ReturnCode = %d, want -1", job.ReturnCode)
	}
}

// TestSubmitJesGetByDSN_Abend classifies an abend (ABAxxx) failure: the spool
// carries an ABA message and a HASP395 ENDED line with no parseable RC, so the
// result reports ErrAba and ReturnCode -1.
func TestSubmitJesGetByDSN_Abend(t *testing.T) {
	s, srv := dialMock(t)
	srv.CompletionReply("RETR",
		"250-When JOB12345 is done, retrieval of its output begins",
		"250 transfer complete")
	spool := "ABA001I TASK ABENDED\n" +
		"$HASP395 MYJOB ENDED\n" +
		" !! END OF JES SPOOL FILE !!\n"
	srv.DataFor("RETR", "", spool)

	job, err := s.SubmitJesGetByDSN("//MYJOB JOB (ACCT)\n//S1 EXEC PGM=IEFBR14\n")
	if !errors.Is(err, zftp.ErrAba) {
		t.Fatalf("err = %v, want ErrAba", err)
	}
	if job.ReturnCode != -1 {
		t.Errorf("job.ReturnCode = %d, want -1", job.ReturnCode)
	}
	if !strings.Contains(job.DisplayName, "MYJOB") {
		t.Errorf("job.DisplayName = %q, want it to contain MYJOB", job.DisplayName)
	}
}
