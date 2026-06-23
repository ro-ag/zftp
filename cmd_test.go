// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"strings"
	"testing"

	zftp "gopkg.in/ro-ag/zftp.v2"
	"gopkg.in/ro-ag/zftp.v2/internal/mockzos"
)

// countVerb counts how many captured command lines start with the given verb,
// tolerating the trailing space the client appends to argument-less commands.
func countVerb(cmds []string, verb string) int {
	n := 0
	for _, c := range cmds {
		if fields := strings.Fields(c); len(fields) > 0 && strings.EqualFold(fields[0], verb) {
			n++
		}
	}
	return n
}

// TestSystem_ReturnsCachedValueWithoutRoundTrip verifies that System() serves the
// value cached during Login and does not issue a second SYST command.
func TestSystem_ReturnsCachedValueWithoutRoundTrip(t *testing.T) {
	s, srv := dialMock(t)

	// Login already issued exactly one SYST to learn the system type.
	systBefore := countVerb(srv.Commands(), "SYST")
	if systBefore != 1 {
		t.Fatalf("precondition: Login issued %d SYST commands, want 1", systBefore)
	}

	got, err := s.System()
	if err != nil {
		t.Fatalf("System() error = %v, want nil", err)
	}
	if got != "MVS" {
		t.Errorf("System() = %q, want %q", got, "MVS")
	}
	if systAfter := countVerb(srv.Commands(), "SYST"); systAfter != systBefore {
		t.Errorf("System() issued %d extra SYST command(s); want the cached value with no round-trip", systAfter-systBefore)
	}
}

// TestSystem_SYSTFailureReturnsErrorWithoutPanic verifies that a failed SYST is
// surfaced as an error rather than a panic across the API boundary. The old
// implementation called panic(err) here.
func TestSystem_SYSTFailureReturnsErrorWithoutPanic(t *testing.T) {
	srv := mockzos.New(t)
	srv.Script("SYST", "500 SYST command not understood")

	// Open without Login so the system type is not cached and System() must hit
	// the network, where SYST fails with a 5xx reply.
	s, err := zftp.Open(srv.Addr())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	got, err := s.System()
	if err == nil {
		t.Fatalf("System() error = nil, want non-nil on SYST failure (got %q)", got)
	}
}
