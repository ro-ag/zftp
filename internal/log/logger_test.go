// SPDX-License-Identifier: Apache-2.0

package log

import (
	"math/bits"
	"testing"
)

// flags lists the four independent logging categories. None and All are derived
// from these, so the tests assert their relationship rather than listing them.
var flags = []Level{ServerLevel, PassiveLevel, CommandLevel, DebugLevel}

// restoreLevel snapshots the package singleton's level and restores it when the
// test finishes. The logger is a process-wide singleton, so without this the
// cases below would leak state into one another (and into the rest of the
// package's tests).
func restoreLevel(t *testing.T) {
	t.Helper()
	saved := Level(std.level.Load())
	t.Cleanup(func() { std.level.Store(uint32(saved)) })
}

// TestLevelsAreDistinctSingleBits is the core ZH06 guard: each category must own
// a single, distinct bit. The original `iota << 1` block produced 2/4/6/8, so
// CommandLevel(6) == ServerLevel(2)|PassiveLevel(4) and the bitmask checks in
// IsEnabled bled across categories.
func TestLevelsAreDistinctSingleBits(t *testing.T) {
	for _, l := range flags {
		if got := bits.OnesCount32(uint32(l)); got != 1 {
			t.Errorf("level %d must be a single bit, has %d bits set", l, got)
		}
	}

	union := ServerLevel | PassiveLevel | CommandLevel | DebugLevel
	if got := bits.OnesCount32(uint32(union)); got != 4 {
		t.Errorf("union of the four levels must occupy 4 distinct bits, has %d (union=%d)", got, union)
	}

	for i := 0; i < len(flags); i++ {
		for j := i + 1; j < len(flags); j++ {
			if flags[i]&flags[j] != 0 {
				t.Errorf("levels %d and %d overlap (AND=%d)", flags[i], flags[j], flags[i]&flags[j])
			}
		}
	}
}

func TestNoneIsZero(t *testing.T) {
	if None != 0 {
		t.Errorf("None must be 0, got %d", None)
	}
}

func TestAllIsUnionOfTheFourLevels(t *testing.T) {
	want := ServerLevel | PassiveLevel | CommandLevel | DebugLevel
	if All != want {
		t.Errorf("All must equal the OR of the four levels (%d), got %d", want, All)
	}
}

// TestSetCommandEnablesOnlyCommand pins the user-visible symptom: selecting one
// category must not silently switch on its neighbours.
func TestSetCommandEnablesOnlyCommand(t *testing.T) {
	restoreLevel(t)
	SetLevel(CommandLevel)

	if !IsEnabled(CommandLevel) {
		t.Error("CommandLevel must be enabled after SetLevel(CommandLevel)")
	}
	for _, l := range []Level{ServerLevel, PassiveLevel, DebugLevel} {
		if IsEnabled(l) {
			t.Errorf("level %d must NOT be enabled by SetLevel(CommandLevel)", l)
		}
	}
}

func TestSetDebugEnablesOnlyDebug(t *testing.T) {
	restoreLevel(t)
	SetLevel(DebugLevel)

	if !IsEnabled(DebugLevel) {
		t.Error("DebugLevel must be enabled after SetLevel(DebugLevel)")
	}
	for _, l := range []Level{ServerLevel, PassiveLevel, CommandLevel} {
		if IsEnabled(l) {
			t.Errorf("level %d must NOT be enabled by SetLevel(DebugLevel)", l)
		}
	}
}

func TestSetAllEnablesEveryLevel(t *testing.T) {
	restoreLevel(t)
	SetLevel(All)

	for _, l := range flags {
		if !IsEnabled(l) {
			t.Errorf("level %d must be enabled after SetLevel(All)", l)
		}
	}
}

func TestSetNoneDisablesEveryLevel(t *testing.T) {
	restoreLevel(t)
	SetLevel(None)

	for _, l := range flags {
		if IsEnabled(l) {
			t.Errorf("level %d must be disabled after SetLevel(None)", l)
		}
	}
}
