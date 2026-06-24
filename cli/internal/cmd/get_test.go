// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	env := map[string]string{
		"ZFTP_HOST": "localhost",
		"ZFTP_USER": "user",
	}

	t.Run("get REMOTE local", func(t *testing.T) {
		fake := &fakeClient{}
		_, err := runCLI(t, fake, env, "get", "REMOTE", "local")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "Get:REMOTE->local"
		if !contains(fake.calls, want) {
			t.Errorf("calls %v does not contain %q", fake.calls, want)
		}
	})

	t.Run("gzip", func(t *testing.T) {
		fake := &fakeClient{}
		_, err := runCLI(t, fake, env, "get", "--gzip", "REMOTE", "local")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "GetAndGzip:REMOTE"
		if !contains(fake.calls, want) {
			t.Errorf("calls %v does not contain %q", fake.calls, want)
		}
	})

	t.Run("offset", func(t *testing.T) {
		fake := &fakeClient{}
		_, err := runCLI(t, fake, env, "get", "--offset", "100", "REMOTE", "local")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "GetAt:REMOTE"
		if !contains(fake.calls, want) {
			t.Errorf("calls %v does not contain %q", fake.calls, want)
		}
	})

	t.Run("ascii+offset rejected before dial", func(t *testing.T) {
		fake := &fakeClient{}
		_, err := runCLI(t, fake, env, "get", "--ascii", "--offset", "100", "REMOTE", "local")
		if err == nil {
			t.Fatal("expected error for --ascii --offset, got nil")
		}
		for _, c := range fake.calls {
			if strings.HasPrefix(c, "GetAt:") {
				t.Errorf("GetAt was called despite rejection: calls=%v", fake.calls)
			}
		}
	})

	t.Run("default local basename", func(t *testing.T) {
		fake := &fakeClient{}
		_, err := runCLI(t, fake, env, "get", "'MY.DATASET'")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "Get:'MY.DATASET'->MY.DATASET"
		if !contains(fake.calls, want) {
			t.Errorf("calls %v does not contain %q", fake.calls, want)
		}
	})
}

func contains(calls []string, want string) bool {
	for _, c := range calls {
		if c == want {
			return true
		}
	}
	return false
}
