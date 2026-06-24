// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"gopkg.in/ro-ag/zftp.v2/internal/log"
	"gopkg.in/ro-ag/zftp.v2/internal/utils"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SetPassiveMode sets the FTP session to passive mode.
// It sends the PASV command to the server and retrieves the response.
// It then parses the response to extract the port number.
// Returns the port number if successful, or an error otherwise.
func (s *FTPSession) SetPassiveMode() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Send PASV command
	response, err := s.SendCommandWithContext(ctx, CodeEnteringPassiveMode, "PASV")
	if err != nil {
		return 0, err
	}

	// Parse the response to find the port
	port, err := findPort(response)
	if err != nil {
		return 0, err
	}
	return port, nil
}

// Parse response to extract port
var regexPort = regexp.MustCompile(`^.*?\(\d+,\d+,\d+,\d+,(\d+),(\d+)\).*$`)

// findPort extracts the port number from the given line of response.
// It uses regular expressions to match the pattern and extract the port numbers.
// Returns the port number if successful, or an error otherwise.
func findPort(line string) (int, error) {

	line = strings.ReplaceAll(line, " ", "")

	matches := regexPort.FindStringSubmatch(line)
	if len(matches) < 3 {
		return 0, fmt.Errorf("cannot find port in text: %s", line)
	}

	// Convert extracted numbers to port
	p1, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}
	p2, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, err
	}
	port := p1<<8 + p2

	return port, nil
}

// childConnection is a passive-mode data connection. It embeds net.Conn — so it
// satisfies the interface the transfer helpers expect, with Read/Write/deadline/
// address methods coming straight from the socket — and overrides Close to make
// teardown idempotent, interrupt an in-flight read, and deregister from the
// session's data-connection map.
//
// No mutex is needed: net.Conn is safe for concurrent Read/Write/Close, the closed
// state is an atomic flag, and Close interrupts a blocked read by pushing a past
// deadline onto the socket before closing it.
type childConnection struct {
	net.Conn
	parent   net.Addr
	maps     *sync.Map
	scan     *scanner
	isClosed atomic.Bool
	lg       *log.Logger
}

// Close tears down the data connection. It is idempotent and safe to call
// concurrently with an in-flight Read/Write/scan, which it interrupts so the
// reader observes the closure promptly.
func (c *childConnection) Close() error {
	caller := utils.Caller()

	// Flip the closed flag first (atomically) so concurrent closes are mutually
	// exclusive and any in-flight scan/read sees the closure immediately.
	if c.isClosed.Swap(true) {
		c.lg.Debugf("<%s> child connection already closed: %s = %p", caller, c.RemoteAddr().String(), c)
		return nil
	}

	// Interrupt any read currently blocked on this connection so it returns
	// promptly; net.Conn permits SetDeadline and Close concurrently with Read/Write.
	_ = c.Conn.SetDeadline(time.Now())

	if _, ok := c.maps.Load(c.RemoteAddr().String()); !ok {
		c.lg.Debugf("<%s>: child connection not present in map: %p", caller, c)
	}

	err := c.Conn.Close()
	if err != nil {
		err = fmt.Errorf("<%s>: %w", caller, err)
	}
	c.maps.Delete(c.RemoteAddr().String())

	c.lg.Debugf("closed child connection: %s = %p", c.RemoteAddr().String(), c)
	return err
}

// IsClosed reports whether the data connection has been closed.
func (c *childConnection) IsClosed() bool {
	return c.isClosed.Load()
}

// ParentAddr returns the address of the parent (control) connection.
func (c *childConnection) ParentAddr() net.Addr {
	return c.parent
}

// Scanner returns the line scanner bound to this connection.
func (c *childConnection) Scanner() *scanner {
	return c.scan
}

// newChildConnection creates a new data connection to the FTP server
// It uses the given port number to connect to the server.
// It uses the TLS configuration if available.
func (s *FTPSession) newChildConnection(port int) (*childConnection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.log.Debugf("attempting to create a new connection with port %d", port)

	address := s.conn.RemoteAddr().String()
	// Split the address into IP/hostname and port
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	host = net.JoinHostPort(host, fmt.Sprintf("%d", port))

	dialer := net.Dialer{Timeout: s.dialCfg.DialTimeout}
	if s.dialCfg.KeepAlivePeriod > 0 {
		dialer.KeepAlive = s.dialCfg.KeepAlivePeriod
	}

	conn, err := dialer.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	if tcp, ok := conn.(*net.TCPConn); ok && s.dialCfg.KeepAlivePeriod > 0 {
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(s.dialCfg.KeepAlivePeriod)
	}

	if s.tlsConfig != nil {
		// Bound the handshake: net.Dialer.Timeout covered only the TCP dial, so a
		// peer that connects but stalls the TLS negotiation would otherwise hang the
		// transfer on the first Read/Write. Fall back to the reply timeout when no
		// dial timeout is configured so the bound is always finite.
		hsTimeout := s.dialCfg.DialTimeout
		if hsTimeout <= 0 {
			hsTimeout = s.dialCfg.replyTimeout()
		}
		tconn, herr := tlsHandshakeBounded(conn, s.tlsConfig, hsTimeout)
		if herr != nil {
			return nil, fmt.Errorf("data-connection TLS handshake: %w", herr)
		}
		conn = tconn
		s.log.Debugf("upgraded connection to TLS")
	}

	child := &childConnection{Conn: conn, parent: conn.RemoteAddr(), maps: &s.dataConns, lg: s.log}
	child.scan = newScanner(conn, child.IsClosed)
	s.dataConns.Store(child.RemoteAddr().String(), child)

	s.log.Debugf("created child connection: %s", child.RemoteAddr().String())
	return child, nil
}

// tlsHandshakeBounded wraps conn in a TLS client and drives the handshake under a
// deadline, so a peer that completes the TCP dial but stalls the TLS negotiation
// cannot hang the caller. The deadline is cleared on success; the data transfer
// manages its own deadlines afterwards. On any failure the TLS connection (and the
// underlying conn it owns) is closed before returning.
func tlsHandshakeBounded(conn net.Conn, cfg *tls.Config, timeout time.Duration) (*tls.Conn, error) {
	tconn := tls.Client(conn, cfg)
	if err := tconn.SetDeadline(time.Now().Add(timeout)); err != nil {
		_ = tconn.Close()
		return nil, err
	}
	if err := tconn.Handshake(); err != nil {
		_ = tconn.Close()
		return nil, err
	}
	if err := tconn.SetDeadline(time.Time{}); err != nil {
		_ = tconn.Close()
		return nil, err
	}
	return tconn, nil
}

// newScanner returns a new scanner for the connection
func newScanner(conn net.Conn, fn func() bool) *scanner {
	return &scanner{scan: bufio.NewScanner(conn), isClosed: fn}
}

// scanner is a wrapper over bufio.Scanner that checks if the connection is closed before scanning.
type scanner struct {
	scan     *bufio.Scanner
	isClosed func() bool
}

// Scan is a wrapper over bufio.Scanner.Scan that checks if the connection is closed before scanning.
func (s *scanner) Scan() bool {
	if s.isClosed() {
		return false
	}
	return s.scan.Scan()
}

// Text is a wrapper over bufio.Scanner.Text that checks if the connection is closed before scanning.
func (s *scanner) Text() string {
	if s.isClosed() {
		return ""
	}
	return s.scan.Text()
}

// Bytes is a wrapper over bufio.Scanner.Bytes that checks if the connection is closed before scanning.
func (s *scanner) Bytes() []byte {
	if s.isClosed() {
		return nil
	}
	return s.scan.Bytes()
}

// Err is a wrapper over bufio.Scanner.
func (s *scanner) Err() error {
	return s.scan.Err()
}

// Split is a wrapper over bufio.Scanner.Split that checks if the connection is closed before scanning.
func (s *scanner) Split(split bufio.SplitFunc) {
	if s.isClosed() {
		return
	}
	s.scan.Split(split)
}

// Buffer is a wrapper over bufio.Scanner.Buffer that checks if the connection is closed before scanning.
func (s *scanner) Buffer(buf []byte, max int) {
	if s.isClosed() {
		return
	}
	s.scan.Buffer(buf, max)
}
