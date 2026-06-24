// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStatCmd_Table(t *testing.T) {
	fake := &fakeClient{system: "MVS is the operating system of this server"}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	out, err := runCLI(t, fake, env, "stat", "-H", "h", "-u", "me")
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	if !strings.Contains(out, "MVS") {
		t.Errorf("output missing system string, got: %s", out)
	}
}

func TestStatCmd_JSON(t *testing.T) {
	fake := &fakeClient{system: "MVS is the operating system of this server"}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	out, err := runCLI(t, fake, env, "stat", "--json", "-H", "h", "-u", "me")
	if err != nil {
		t.Fatalf("stat --json error: %v", err)
	}
	var obj map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &obj); jsonErr != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", jsonErr, out)
	}
	if _, ok := obj["system"]; !ok {
		t.Errorf("JSON output missing 'system' key, got: %s", out)
	}
}
