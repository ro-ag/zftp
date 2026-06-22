// SPDX-License-Identifier: Apache-2.0

package hfs

import (
	"encoding/json"
	"testing"
)

// TestFieldInt_OverflowDistinctFromRealMax pins the core ZH04 bug: a z/OS
// display-overflow indicator ("+++++") must never be confused with a legitimate
// value of exactly 65535. Both appear in the real dataset fixture and both used
// to render as "65535"/Value()==65535.
func TestFieldInt_OverflowDistinctFromRealMax(t *testing.T) {
	var real, overflow FieldInt
	if err := real.parse("65535"); err != nil {
		t.Fatalf("parse(65535): %v", err)
	}
	if err := overflow.parse("+++++"); err != nil {
		t.Fatalf("parse(+++++): %v", err)
	}

	// The real value is a real value.
	if real.IsOverflow() {
		t.Error("real 65535 reported IsOverflow()==true")
	}
	if real.Value() != 65535 {
		t.Errorf("real Value()=%d, want 65535", real.Value())
	}
	if real.String() != "65535" {
		t.Errorf("real String()=%q, want \"65535\"", real.String())
	}

	// The overflow indicator is flagged out of band.
	if !overflow.IsOverflow() {
		t.Error("overflow \"+++++\" reported IsOverflow()==false")
	}

	// And the two are distinguishable across every observable surface.
	if overflow.String() == real.String() {
		t.Errorf("overflow String()=%q indistinguishable from real %q", overflow.String(), real.String())
	}
	if overflow.Value() == real.Value() && !real.IsOverflow() && overflow.IsOverflow() {
		// Value() may legitimately be 0 for overflow; that is fine *because*
		// IsOverflow() disambiguates. This guard only fails if NOTHING
		// distinguishes them, which is the bug.
	}
	realJSON := mustMarshal(t, &real)
	overflowJSON := mustMarshal(t, &overflow)
	if realJSON == overflowJSON {
		t.Errorf("overflow JSON %s indistinguishable from real JSON %s", overflowJSON, realJSON)
	}
	if realJSON != "65535" {
		t.Errorf("real JSON=%s, want 65535 (a real number, not a magic marker)", realJSON)
	}
}

// TestFieldInt_NoSilentWrap pins the second ZH04 bug: values that fit the
// fixed-width column but exceed uint16 used to be silently truncated by
// uint16(value) (65536 -> 0, 70000 -> 4464). They must round-trip exactly.
func TestFieldInt_NoSilentWrap(t *testing.T) {
	cases := []struct {
		in   string
		want uint32
	}{
		{"70000", 70000},  // 5-char Used column; uint16 wrapped to 4464
		{"65536", 65536},  // uint16 wrapped to 0, rendered as ""
		{"999999", 999999}, // 6-char Lrecl/BlkSz column max
	}
	for _, tc := range cases {
		var f FieldInt
		if err := f.parse(tc.in); err != nil {
			t.Fatalf("parse(%q): unexpected error %v", tc.in, err)
		}
		if f.IsOverflow() {
			t.Errorf("parse(%q): IsOverflow()==true, want a real value", tc.in)
		}
		if f.Value() != tc.want {
			t.Errorf("parse(%q): Value()=%d, want %d (no wrap)", tc.in, f.Value(), tc.want)
		}
		if f.String() != tc.in {
			t.Errorf("parse(%q): String()=%q, want %q", tc.in, f.String(), tc.in)
		}
	}
}

// TestFieldInt_OutOfRangeErrors verifies that a genuinely out-of-range or
// malformed numeric value returns an error instead of wrapping.
func TestFieldInt_OutOfRangeErrors(t *testing.T) {
	for _, in := range []string{
		"4294967296", // one past uint32 max
		"99999999999",
		"-1",  // counts are non-negative; a leading '-' must not wrap
		"12x3", // non-numeric
	} {
		var f FieldInt
		if err := f.parse(in); err == nil {
			t.Errorf("parse(%q): want error, got Value()=%d", in, f.Value())
		}
	}
}

// TestFieldInt_EmptyAndZero confirms the absent/zero cases are unchanged and
// stay distinct from overflow.
func TestFieldInt_EmptyAndZero(t *testing.T) {
	for _, in := range []string{"", "   ", "0"} {
		var f FieldInt
		if err := f.parse(in); err != nil {
			t.Fatalf("parse(%q): %v", in, err)
		}
		if f.IsOverflow() {
			t.Errorf("parse(%q): IsOverflow()==true, want false", in)
		}
		if f.Value() != 0 {
			t.Errorf("parse(%q): Value()=%d, want 0", in, f.Value())
		}
		if f.String() != "" {
			t.Errorf("parse(%q): String()=%q, want \"\"", in, f.String())
		}
		if got := mustMarshal(t, &f); got != "null" {
			t.Errorf("parse(%q): JSON=%s, want null", in, got)
		}
	}
}

// TestFieldInt_JSONRoundTrip ensures every representation survives a
// marshal/unmarshal cycle, including the overflow marker.
func TestFieldInt_JSONRoundTrip(t *testing.T) {
	for _, in := range []string{"", "0", "65535", "70000", "999999", "+++++"} {
		var f FieldInt
		if err := f.parse(in); err != nil {
			t.Fatalf("parse(%q): %v", in, err)
		}
		b, err := json.Marshal(&f)
		if err != nil {
			t.Fatalf("marshal(%q): %v", in, err)
		}
		var g FieldInt
		if err := json.Unmarshal(b, &g); err != nil {
			t.Fatalf("unmarshal(%q) from %s: %v", in, b, err)
		}
		if g.IsOverflow() != f.IsOverflow() || g.Value() != f.Value() {
			t.Errorf("round-trip %q via %s: got {overflow:%v value:%d}, want {overflow:%v value:%d}",
				in, b, g.IsOverflow(), g.Value(), f.IsOverflow(), f.Value())
		}
	}
}

func mustMarshal(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}
