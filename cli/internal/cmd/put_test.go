// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"strings"
	"testing"
)

func TestPut(t *testing.T) {
	env := map[string]string{
		"ZFTP_HOST": "localhost",
		"ZFTP_USER": "user",
	}

	t.Run("put local remote", func(t *testing.T) {
		fake := &fakeClient{}
		_, err := runCLI(t, fake, env, "put", "local", "REMOTE")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "Put:local->REMOTE"
		if !contains(fake.calls, want) {
			t.Errorf("calls %v does not contain %q", fake.calls, want)
		}
	})

	t.Run("offset", func(t *testing.T) {
		fake := &fakeClient{}
		_, err := runCLI(t, fake, env, "put", "--offset", "100", "local", "REMOTE")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "PutAt:local"
		if !contains(fake.calls, want) {
			t.Errorf("calls %v does not contain %q", fake.calls, want)
		}
	})

	t.Run("ascii+offset rejected before dial", func(t *testing.T) {
		fake := &fakeClient{}
		_, err := runCLI(t, fake, env, "put", "--ascii", "--offset", "100", "local", "REMOTE")
		if err == nil {
			t.Fatal("expected error for --ascii --offset, got nil")
		}
		for _, c := range fake.calls {
			if strings.HasPrefix(c, "PutAt:") {
				t.Errorf("PutAt was called despite rejection: calls=%v", fake.calls)
			}
		}
	})

	t.Run("default remote basename", func(t *testing.T) {
		fake := &fakeClient{}
		_, err := runCLI(t, fake, env, "put", "/some/path/myfile.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "Put:/some/path/myfile.txt->myfile.txt"
		if !contains(fake.calls, want) {
			t.Errorf("calls %v does not contain %q", fake.calls, want)
		}
	})
}
