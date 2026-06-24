// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"gopkg.in/ro-ag/zftp.v2/hfs"
)

// TestLsCmd_PDS verifies the --pds flag calls ListPds and renders the member name.
func TestLsCmd_PDS(t *testing.T) {
	// "ARTIST    01.00 2021/08/20 2021/08/20 07:51     6     6     0 A99993"
	// is 69 chars — at or past idStart (61), so statistics are parsed.
	m, err := hfs.ParseInfoPdsMember("COBOL01   01.00 2024/06/20 2024/06/20 10:30   100   100     0 USER001")
	if err != nil {
		t.Fatalf("ParseInfoPdsMember: %v", err)
	}
	fake := &fakeClient{pds: []hfs.InfoPdsMember{m}}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	out, err := runCLI(t, fake, env, "ls", "X.*", "--pds", "-H", "h", "-u", "me")
	if err != nil {
		t.Fatalf("ls --pds error: %v", err)
	}
	if !strings.Contains(out, "COBOL01") {
		t.Errorf("output missing member name, got: %s", out)
	}
	if !contains(fake.calls, "ListPds:X.*") {
		t.Errorf("ListPds:X.* not in calls %v", fake.calls)
	}
}

// TestLsCmd_HFS verifies the --hfs flag calls List and prints each line.
func TestLsCmd_HFS(t *testing.T) {
	fake := &fakeClient{listLines: []string{"/u/me/file1", "/u/me/file2"}}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	out, err := runCLI(t, fake, env, "ls", "/u/me/*", "--hfs", "-H", "h", "-u", "me")
	if err != nil {
		t.Fatalf("ls --hfs error: %v", err)
	}
	if !strings.Contains(out, "/u/me/file1") {
		t.Errorf("output missing /u/me/file1, got: %s", out)
	}
	if !strings.Contains(out, "/u/me/file2") {
		t.Errorf("output missing /u/me/file2, got: %s", out)
	}
	if !contains(fake.calls, "List:/u/me/*") {
		t.Errorf("List:/u/me/* not in calls %v", fake.calls)
	}
}

// TestLsCmd_Error verifies ls propagates a client error.
func TestLsCmd_Error(t *testing.T) {
	fake := &fakeClient{err: errors.New("boom")}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	_, err := runCLI(t, fake, env, "ls", "X.*", "-H", "h", "-u", "me")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestGetCmd_ASCII verifies --ascii exercises the ascii branch of transferType.
func TestGetCmd_ASCII(t *testing.T) {
	fake := &fakeClient{}
	env := map[string]string{"ZFTP_HOST": "h", "ZFTP_USER": "me", "ZFTP_PASSWORD": "pw"}
	_, err := runCLI(t, fake, env, "get", "REMOTE", "local", "--ascii")
	if err != nil {
		t.Fatalf("get --ascii error: %v", err)
	}
	if !contains(fake.calls, "Get:REMOTE->local") {
		t.Errorf("Get:REMOTE->local not in calls %v", fake.calls)
	}
}

// TestStatCmd_Error verifies stat propagates a System() error.
func TestStatCmd_Error(t *testing.T) {
	fake := &fakeClient{err: errors.New("syst failed"), system: ""}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	_, err := runCLI(t, fake, env, "stat", "-H", "h", "-u", "me")
	if err == nil {
		t.Fatal("expected error from stat, got nil")
	}
	if !strings.Contains(err.Error(), "syst failed") {
		t.Errorf("expected 'syst failed' in error, got: %v", err)
	}
}

// TestDial_PasswordPromptError verifies that a prompt failure is wrapped and returned
// before the connect func is called.
func TestDial_PasswordPromptError(t *testing.T) {
	var out, errOut bytes.Buffer
	d := deps{
		connect: func(connOpts) (client, error) {
			return nil, errors.New("should not be called")
		},
		getenv: func(string) string { return "" }, // no ZFTP_PASSWORD
		prompt: func() (string, error) { return "", errors.New("no tty") },
		out:    &out,
		errOut: &errOut,
	}
	root := newRootCmd(d, BuildInfo{Version: "test"})
	root.SetArgs([]string{"ls", "X.*", "-H", "h", "-u", "me"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error from prompt failure, got nil")
	}
	if !strings.Contains(err.Error(), "no tty") {
		t.Errorf("expected 'no tty' in error, got: %v", err)
	}
}

// TestWithClient_DialConnectError verifies that a connect() failure is returned
// from withClient without calling the wrapped function.
func TestWithClient_DialConnectError(t *testing.T) {
	var out, errOut bytes.Buffer
	d := deps{
		connect: func(connOpts) (client, error) { return nil, errors.New("dial fail") },
		getenv:  func(k string) string { m := map[string]string{"ZFTP_PASSWORD": "pw"}; return m[k] },
		prompt:  func() (string, error) { return "pw", nil },
		out:     &out,
		errOut:  &errOut,
	}
	root := newRootCmd(d, BuildInfo{Version: "test"})
	root.SetArgs([]string{"rm", "USER.A", "-H", "h", "-u", "me"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected dial error, got nil")
	}
	if !strings.Contains(err.Error(), "dial fail") {
		t.Errorf("expected 'dial fail' in error, got: %v", err)
	}
}

// TestJobCmd_JSON verifies --json output from the job command decodes and contains a JobId.
func TestJobCmd_JSON(t *testing.T) {
	fake := &fakeClient{jobDetail: makeInfoJobDetail(t)}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	out, err := runCLI(t, fake, env, "job", "JOB12345", "--json", "-H", "h", "-u", "me")
	if err != nil {
		t.Fatalf("job --json error: %v", err)
	}
	var obj map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &obj); jsonErr != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", jsonErr, out)
	}
	jobID, ok := obj["JobId"]
	if !ok {
		t.Errorf("JSON output missing 'JobId' key, got keys: %v, output: %s", obj, out)
	}
	if jobID == "" || jobID == nil {
		t.Errorf("expected non-empty JobId, got: %v", jobID)
	}
}
