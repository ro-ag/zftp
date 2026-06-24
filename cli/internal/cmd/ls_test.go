// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/ro-ag/zftp.v2/hfs"
)

// makeDataset builds an hfs.InfoDataset fixture via JSON unmarshal so the
// unexported FieldString/FieldInt internals are populated correctly.
func makeDataset(t *testing.T) hfs.InfoDataset {
	t.Helper()
	var ds hfs.InfoDataset
	if err := json.Unmarshal([]byte(`{"Dsname":"USER.DATA","Volume":"VOL001","Recfm":"FB","Lrecl":80,"Dsorg":"PS"}`), &ds); err != nil {
		t.Fatalf("makeDataset: %v", err)
	}
	return ds
}

// TestLsCmd_Table verifies the table output contains the header and the dataset name.
func TestLsCmd_Table(t *testing.T) {
	fake := &fakeClient{
		datasets: []hfs.InfoDataset{makeDataset(t)},
	}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	out, err := runCLI(t, fake, env, "ls", "USER.*", "-H", "h", "-u", "me")
	if err != nil {
		t.Fatalf("ls error: %v", err)
	}
	if !strings.Contains(out, "NAME") {
		t.Errorf("output missing table header, got: %s", out)
	}
	if !strings.Contains(out, "USER.DATA") {
		t.Errorf("output missing dataset name, got: %s", out)
	}
	// Verify ListDatasets was called with the pattern argument.
	found := false
	for _, c := range fake.calls {
		if c == "ListDatasets:USER.*" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListDatasets:USER.* not in calls %v", fake.calls)
	}
}

// TestLsCmd_JSON verifies the JSON output decodes to a non-empty array and that
// Close was called (deferred cleanup).
func TestLsCmd_JSON(t *testing.T) {
	fake := &fakeClient{
		datasets: []hfs.InfoDataset{makeDataset(t)},
	}
	env := map[string]string{"ZFTP_PASSWORD": "pw"}
	out, err := runCLI(t, fake, env, "ls", "USER.*", "-H", "h", "-u", "me", "--json")
	if err != nil {
		t.Fatalf("ls --json error: %v", err)
	}
	var arr []json.RawMessage
	if jsonErr := json.Unmarshal([]byte(out), &arr); jsonErr != nil {
		t.Fatalf("output is not valid JSON array: %v\noutput: %s", jsonErr, out)
	}
	if len(arr) == 0 {
		t.Errorf("expected non-empty JSON array, got: %s", out)
	}
	// Verify Close was called.
	found := false
	for _, c := range fake.calls {
		if c == "Close" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Close not in calls %v", fake.calls)
	}
}
