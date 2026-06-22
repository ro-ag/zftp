// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"testing"
)

func TestSetPassiveMode_Valid(t *testing.T) {
	s, srv := dialMock(t)
	// Override the real PASV handler with a fixed advertisement: port = 4*256+210.
	srv.Script("PASV", "227 Entering Passive Mode (10,1,2,3,4,210)")

	port, err := s.SetPassiveMode()
	if err != nil {
		t.Fatalf("SetPassiveMode: %v", err)
	}
	if want := 4*256 + 210; port != want {
		t.Errorf("port = %d, want %d", port, want)
	}
}

func TestSetPassiveMode_Malformed(t *testing.T) {
	s, srv := dialMock(t)
	srv.Script("PASV", "227 Entering Passive Mode but with no address tuple")

	if _, err := s.SetPassiveMode(); err == nil {
		t.Fatal("want error for malformed 227 reply")
	}
}
