package zftp

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"gopkg.in/ro-ag/zftp.v1/internal/log"
	"gopkg.in/ro-ag/zftp.v1/internal/utils"
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
	if matches == nil || len(matches) < 3 {
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

// connectDataConn is a wrapper over net.Dial that sets a timeout on the connection.
// its has mutex control to allow graceful closing of the connection.
type childConnection struct {
	conn     net.Conn
	parent   net.Addr
	maps     *sync.Map
	scan     *scanner
	mu       sync.RWMutex
	isClosed atomic.Bool
}

// Read is a wrapper over net.Conn.Read that sets a timeout on the connection.
func (c *childConnection) Read(b []byte) (n int, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn.Read(b)
}

// Write is a wrapper over net.Conn.Write that sets a timeout on the connection.
func (c *childConnection) Write(b []byte) (n int, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn.Write(b)
}

// Close closes the connection.
// It acquires an exclusive lock to ensure exclusive access to the connection.
// It checks the isClosed flag to avoid closing the connection multiple times.
// It updates the isClosed flag and deletes the connection from the maps.
// Returns an error if closing the connection fails.
func (c *childConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	alreadyClosed := c.isClosed.Load()
	caller := utils.Caller()

	log.Debugf("<%s> attempting to close child connection: %s = %p | closed=%v", caller, c.RemoteAddr().String(), c, alreadyClosed)

	if c.isClosed.Load() {
		return nil
	}

	x, ok := c.maps.Load(c.RemoteAddr().String())
	if !ok {
		log.Panicf("<%s>: cannot find child connection in map: %p | %p", caller, c, x)
	}

	err := c.conn.Close()
	if err != nil {
		err = fmt.Errorf("<%s>: %w", caller, err)
	}

	c.isClosed.Store(true)
	c.maps.Delete(c.RemoteAddr().String())

	log.Debugf("closed child connection: %s = %p | closed=%v", c.RemoteAddr().String(), c, c.isClosed.Load())

	return err
}

// IsClosed returns true if the connection is closed, false otherwise.
// its uses atomic operations to read the isClosed flag.
func (c *childConnection) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isClosed.Load()
}

// LocalAddr wraps net.Conn.LocalAddr
func (c *childConnection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr wraps net.Conn.RemoteAddr
func (c *childConnection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// ParentAddr returns the address of the parent connection.
func (c *childConnection) ParentAddr() net.Addr {
	return c.parent
}

// SetDeadline wraps net.Conn.SetDeadline
func (c *childConnection) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline wraps net.Conn.SetReadDeadline
func (c *childConnection) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline wraps net.Conn.SetWriteDeadline
func (c *childConnection) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// Scanner returns a new scanner for the connection
func (c *childConnection) Scanner() *scanner {
	return c.scan
}

// newChildConnection creates a new data connection to the FTP server
// It uses the given port number to connect to the server.
// It uses the TLS configuration if available.
func (s *FTPSession) newChildConnection(port int) (*childConnection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Debugf("attempting to create a new connection with port %d", port)

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
		conn = tls.Client(conn, s.tlsConfig)
		log.Debugf("upgraded connection to TLS")
	}

	child := childConnection{conn: conn, parent: conn.RemoteAddr(), maps: &s.dataConns}
	child.scan = newScanner(conn, child.IsClosed)
	s.dataConns.Store(child.RemoteAddr().String(), &child)

	log.Debugf("created child connection: %s", child.RemoteAddr().String())
	return &child, nil
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
