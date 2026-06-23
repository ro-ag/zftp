// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"testing"
	"time"
)

// TestSetKeepAlive_OnTCPConn verifies SetKeepAlive succeeds on the loopback TCP
// control connection the mock provides (enabling with a period, then disabling).
func TestSetKeepAlive_OnTCPConn(t *testing.T) {
	s, _ := dialMock(t)

	if err := s.SetKeepAlive(30 * time.Second); err != nil {
		t.Fatalf("SetKeepAlive(30s): %v", err)
	}
	if err := s.SetKeepAlive(0); err != nil {
		t.Fatalf("SetKeepAlive(0) (disable): %v", err)
	}
}

// TestAddrAccessors verifies the address accessors report the underlying socket's
// addresses: RemoteAddr matches the server the client dialed and LocalAddr is the
// non-nil client side. They replace the removed Conn() footgun without exposing
// Read/Write/Close.
func TestAddrAccessors(t *testing.T) {
	s, srv := dialMock(t)

	ra := s.RemoteAddr()
	if ra == nil {
		t.Fatal("RemoteAddr() = nil")
	}
	if ra.String() != srv.Addr() {
		t.Errorf("RemoteAddr() = %q, want %q", ra.String(), srv.Addr())
	}

	la := s.LocalAddr()
	if la == nil {
		t.Fatal("LocalAddr() = nil")
	}
	if la.Network() != "tcp" {
		t.Errorf("LocalAddr().Network() = %q, want tcp", la.Network())
	}
}
