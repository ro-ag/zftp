// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"crypto/tls"
	"io"
	"net"
	"testing"
	"time"
)

// M3: the data-connection TLS handshake must be bounded. A peer that completes
// the TCP dial but never speaks TLS (stalled ServerHello) must make the handshake
// fail on a deadline rather than hang the transfer forever.
func TestTLSHandshakeBounded_StalledServerTimesOut(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		io.Copy(io.Discard, c) // consume ClientHello, never reply → stalls handshake
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	done := make(chan error, 1)
	go func() {
		_, e := tlsHandshakeBounded(conn, &tls.Config{InsecureSkipVerify: true}, 200*time.Millisecond)
		done <- e
	}()
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("expected a TLS handshake timeout error, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("TLS handshake hung: no deadline on the data-connection handshake")
	}
}
