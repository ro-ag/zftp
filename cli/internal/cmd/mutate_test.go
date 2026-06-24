// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"errors"
	"strings"
	"testing"
)

func TestRmCmd(t *testing.T) {
	fake := &fakeClient{}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	_, err := runCLI(t, fake, env, "rm", "USER.A", "-H", "h", "-u", "me")
	if err != nil {
		t.Fatalf("rm error: %v", err)
	}
	want := "Delete:USER.A"
	found := false
	for _, c := range fake.calls {
		if c == want {
			found = true
		}
	}
	if !found {
		t.Errorf("%s not in calls %v", want, fake.calls)
	}
	foundClose := false
	for _, c := range fake.calls {
		if c == "Close" {
			foundClose = true
		}
	}
	if !foundClose {
		t.Errorf("Close not in calls %v", fake.calls)
	}
}

func TestMkdirCmd(t *testing.T) {
	fake := &fakeClient{}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	_, err := runCLI(t, fake, env, "mkdir", "/u/x", "-H", "h", "-u", "me")
	if err != nil {
		t.Fatalf("mkdir error: %v", err)
	}
	want := "Mkdir:/u/x"
	found := false
	for _, c := range fake.calls {
		if c == want {
			found = true
		}
	}
	if !found {
		t.Errorf("%s not in calls %v", want, fake.calls)
	}
}

func TestMvCmd(t *testing.T) {
	fake := &fakeClient{}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	_, err := runCLI(t, fake, env, "mv", "A", "B", "-H", "h", "-u", "me")
	if err != nil {
		t.Fatalf("mv error: %v", err)
	}
	want := "Rename:A->B"
	found := false
	for _, c := range fake.calls {
		if c == want {
			found = true
		}
	}
	if !found {
		t.Errorf("%s not in calls %v", want, fake.calls)
	}
}

func TestChmodCmd(t *testing.T) {
	fake := &fakeClient{}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	_, err := runCLI(t, fake, env, "chmod", "750", "/u/f", "-H", "h", "-u", "me")
	if err != nil {
		t.Fatalf("chmod error: %v", err)
	}
	want := "Chmod:750 /u/f"
	found := false
	for _, c := range fake.calls {
		if c == want {
			found = true
		}
	}
	if !found {
		t.Errorf("%s not in calls %v", want, fake.calls)
	}
}

func TestMutateCmd_Error(t *testing.T) {
	fake := &fakeClient{err: errors.New("denied")}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	_, err := runCLI(t, fake, env, "rm", "USER.A", "-H", "h", "-u", "me")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "denied") {
		t.Errorf("expected 'denied' in error, got: %v", err)
	}
}
