// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/ro-ag/zftp.v2"
	"gopkg.in/ro-ag/zftp.v2/hfs"
)

func TestSubmitCmd_Table(t *testing.T) {
	fake := &fakeClient{submitJob: &zftp.JesJob{ID: "JOB12345"}}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	out, err := runCLI(t, fake, env, "submit", "job.jcl", "-H", "h", "-u", "me")
	if err != nil {
		t.Fatalf("submit error: %v", err)
	}
	if !strings.Contains(out, "JOB12345") {
		t.Errorf("output missing job ID, got: %s", out)
	}
	found := false
	for _, c := range fake.calls {
		if c == "SubmitJCLFile:job.jcl" {
			found = true
		}
	}
	if !found {
		t.Errorf("SubmitJCLFile:job.jcl not in calls %v", fake.calls)
	}
}

func TestSubmitCmd_JSON(t *testing.T) {
	fake := &fakeClient{submitJob: &zftp.JesJob{ID: "JOB12345"}}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	out, err := runCLI(t, fake, env, "submit", "--json", "job.jcl", "-H", "h", "-u", "me")
	if err != nil {
		t.Fatalf("submit --json error: %v", err)
	}
	var obj map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &obj); jsonErr != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", jsonErr, out)
	}
	// JesJob has no json tags so fields marshal as their Go name (uppercase)
	if _, ok := obj["ID"]; !ok {
		t.Errorf("JSON output missing 'ID' key, got keys: %v, output: %s", obj, out)
	}
	if obj["ID"] != "JOB12345" {
		t.Errorf("expected ID=JOB12345, got: %v", obj["ID"])
	}
}

func makeInfoJobs(t *testing.T) []hfs.InfoJob {
	t.Helper()
	lines := []string{
		"JOBNAME  JOBID    OWNER    STATUS CLASS",
		"MYJOB    JOB12345 USER     OUTPUT A",
	}
	jobs, err := hfs.ParseInfoJob(lines)
	if err != nil {
		t.Fatalf("ParseInfoJob: %v", err)
	}
	return jobs
}

func makeInfoJobDetail(t *testing.T) *hfs.InfoJobDetail {
	t.Helper()
	lines := []string{
		"JOBNAME  JOBID    OWNER    STATUS CLASS",
		"MYJOB    JOB12345 USER     OUTPUT A        RC=0000",
	}
	jd, err := hfs.ParseInfoJobDetail(lines)
	if err != nil {
		t.Fatalf("ParseInfoJobDetail: %v", err)
	}
	return jd
}

func TestJobsCmd_Table(t *testing.T) {
	fake := &fakeClient{jobs: makeInfoJobs(t)}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	out, err := runCLI(t, fake, env, "jobs", "*", "-H", "h", "-u", "me")
	if err != nil {
		t.Fatalf("jobs error: %v", err)
	}
	if !strings.Contains(out, "MYJOB") {
		t.Errorf("output missing job name, got: %s", out)
	}
	found := false
	for _, c := range fake.calls {
		if c == "ListSpool:*" {
			found = true
		}
	}
	if !found {
		t.Errorf("ListSpool:* not in calls %v", fake.calls)
	}
}

func TestJobCmd_Detail(t *testing.T) {
	fake := &fakeClient{jobDetail: makeInfoJobDetail(t)}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	_, err := runCLI(t, fake, env, "job", "JOB12345", "-H", "h", "-u", "me")
	if err != nil {
		t.Fatalf("job error: %v", err)
	}
	found := false
	for _, c := range fake.calls {
		if c == "GetJobStatus:JOB12345" {
			found = true
		}
	}
	if !found {
		t.Errorf("GetJobStatus:JOB12345 not in calls %v", fake.calls)
	}
}

func TestJobCmd_Purge(t *testing.T) {
	fake := &fakeClient{}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	_, err := runCLI(t, fake, env, "job", "purge", "JOB12345", "-H", "h", "-u", "me")
	if err != nil {
		t.Fatalf("job purge error: %v", err)
	}
	found := false
	for _, c := range fake.calls {
		if c == "PurgeJob:JOB12345" {
			found = true
		}
	}
	if !found {
		t.Errorf("PurgeJob:JOB12345 not in calls %v", fake.calls)
	}
}
