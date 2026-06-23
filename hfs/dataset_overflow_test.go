// SPDX-License-Identifier: Apache-2.0

package hfs_test

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"gopkg.in/ro-ag/zftp.v2/hfs"
)

// TestParseInfoDataset_OverflowVsRealMax is the end-to-end ZH04 guard. The real
// fixture contains both a legitimate Used==65535 row and a display-overflow
// ("+++++") row. Parsed through the public API they must remain distinguishable:
// the overflow row reports IsOverflow()==true and never masquerades as 65535.
func TestParseInfoDataset_OverflowVsRealMax(t *testing.T) {
	const (
		realName     = "ABCD.EF.HHHH4.P12.D190909.T0803.DEV" // Used column == "65535"
		overflowName = "CNDM.DBNZF0.EXPSTS.G0118V00"         // Used column == "+++++"
	)

	f, err := os.Open("dataset_test.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var real, overflow *hfs.InfoDataset
	sc := bufio.NewScanner(f)
	first := true
	for sc.Scan() {
		if first {
			first = false
			continue
		}
		d, err := modernParser.Parse(sc.Text())
		if err != nil {
			t.Fatalf("ParseInfoDataset(%q): %v", sc.Text(), err)
		}
		switch d.Name() {
		case realName:
			dd := d
			real = &dd
		case overflowName:
			dd := d
			overflow = &dd
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if real == nil || overflow == nil {
		t.Fatalf("fixture rows missing: real=%v overflow=%v", real != nil, overflow != nil)
	}

	// Real 65535 is a real number.
	if real.Used.IsOverflow() {
		t.Errorf("%s: Used.IsOverflow()==true, want false", realName)
	}
	if real.Used.Value() != 65535 {
		t.Errorf("%s: Used.Value()=%d, want 65535", realName, real.Used.Value())
	}
	if real.Used.String() != "65535" {
		t.Errorf("%s: Used.String()=%q, want \"65535\"", realName, real.Used.String())
	}

	// Overflow is flagged and does NOT render as the magic 65535.
	if !overflow.Used.IsOverflow() {
		t.Errorf("%s: Used.IsOverflow()==false, want true", overflowName)
	}
	if overflow.Used.String() == "65535" {
		t.Errorf("%s: Used.String()==\"65535\", overflow masquerading as real max", overflowName)
	}
	if overflow.Used.String() == real.Used.String() {
		t.Errorf("overflow and real-max Used.String() both %q — indistinguishable",
			real.Used.String())
	}
	if !strings.Contains(overflow.Used.String(), "+") {
		t.Errorf("%s: Used.String()=%q, want a clearly-marked overflow", overflowName, overflow.Used.String())
	}
}
