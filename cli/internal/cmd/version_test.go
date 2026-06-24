// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// TestVersionCmd_Plain verifies the plain-text output contains the version string.
func TestVersionCmd_Plain(t *testing.T) {
	var buf bytes.Buffer
	d := deps{
		connect: nil,
		getenv:  func(string) string { return "" },
		prompt:  func() (string, error) { return "", nil },
		out:     &buf,
		errOut:  &buf,
	}
	root := newRootCmd(d, BuildInfo{Version: "2.0.0", Commit: "abc", Date: "d"})
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(buf.String(), "2.0.0") {
		t.Fatalf("plain output %q does not contain 2.0.0", buf.String())
	}
}

// TestVersionCmd_JSON verifies the JSON output contains the Version field with the correct value.
func TestVersionCmd_JSON(t *testing.T) {
	var buf bytes.Buffer
	d := deps{
		connect: nil,
		getenv:  func(string) string { return "" },
		prompt:  func() (string, error) { return "", nil },
		out:     &buf,
		errOut:  &buf,
	}
	root := newRootCmd(d, BuildInfo{Version: "2.0.0", Commit: "abc", Date: "d"})
	root.SetArgs([]string{"version", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, `"Version"`) {
		t.Fatalf("JSON output %q does not contain key \"Version\"", got)
	}
	if !strings.Contains(got, "2.0.0") {
		t.Fatalf("JSON output %q does not contain 2.0.0", got)
	}
}
