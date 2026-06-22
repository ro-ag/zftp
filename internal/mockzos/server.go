// SPDX-License-Identifier: Apache-2.0

// Package mockzos provides a lightweight, in-process FTP server that emulates
// the subset of the z/OS FTP dialect the zftp client relies on: a 220 greeting,
// USER/PASS login, SYST reporting MVS, TYPE/SITE/XSTA replies, passive-mode data
// connections, and LIST/NLST/RETR/STOR transfers. It lets the real client be
// exercised end-to-end (dial, passive negotiation, data transfer, multiline
// reply parsing) over loopback with no mainframe.
//
// It is a test helper and lives under internal/ so it never becomes part of the
// public API and cannot create an import cycle with the library under test.
package mockzos

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// dataTimeout bounds how long the server waits for the client to open a passive
// data connection, so a misbehaving test fails fast instead of hanging.
const dataTimeout = 10 * time.Second

// Server is an in-process FTP server bound to a loopback ephemeral port.
type Server struct {
	tb     testing.TB
	ln     net.Listener
	closed atomic.Bool
	wg     sync.WaitGroup

	mu          sync.Mutex
	lineScripts map[string][]string // full command line (upper) -> raw reply lines
	verbScripts map[string][]string // verb (upper) -> raw reply lines
	dataByLine  map[string]string   // full command line (upper) -> payload to send
	dataByVerb  map[string]string   // verb (upper) -> payload to send
	stored      map[string][]byte   // STOR arg (upper) -> captured payload
	received    []string            // every command line received, in order
}

// New starts a Server on 127.0.0.1:0 and registers cleanup with the test.
func New(tb testing.TB) *Server {
	tb.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatalf("mockzos listen: %v", err)
	}
	s := &Server{
		tb:          tb,
		ln:          ln,
		lineScripts: map[string][]string{},
		verbScripts: map[string][]string{},
		dataByLine:  map[string]string{},
		dataByVerb:  map[string]string{},
		stored:      map[string][]byte{},
	}
	s.wg.Add(1)
	go s.serve()
	tb.Cleanup(s.Close)
	return s
}

// Addr returns the host:port the server is listening on.
func (s *Server) Addr() string { return s.ln.Addr().String() }

// Close stops the server and waits for in-flight connections to drain.
func (s *Server) Close() {
	if s.closed.Swap(true) {
		return
	}
	_ = s.ln.Close()
	s.wg.Wait()
}

func (s *Server) serve() {
	defer s.wg.Done()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return // listener closed
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handle(conn)
		}()
	}
}

// session holds per-connection state (the pending passive data listener).
type session struct {
	conn net.Conn
	pasv net.Listener
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	sess := &session{conn: conn}
	defer func() {
		if sess.pasv != nil {
			_ = sess.pasv.Close()
		}
	}()

	writeLines(conn, []string{"220 mockzos FTP service ready"})

	r := bufio.NewReader(conn)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		s.mu.Lock()
		s.received = append(s.received, line)
		s.mu.Unlock()
		verb, arg := splitCommand(line)
		if s.dispatch(sess, line, verb, arg) {
			return // QUIT
		}
	}
}

// dispatch handles a single command. It returns true when the connection should
// close (QUIT).
func (s *Server) dispatch(sess *session, line, verb, arg string) bool {
	// Explicit scripted replies take precedence over defaults.
	if reply, ok := s.scriptFor(line, verb); ok {
		writeLines(sess.conn, reply)
		return verb == "QUIT"
	}

	switch verb {
	case "USER":
		writeLines(sess.conn, []string{"331 send password"})
	case "PASS":
		writeLines(sess.conn, []string{"230 user logged in, proceed"})
	case "SYST":
		writeLines(sess.conn, []string{"215 MVS is the operating system of this server. FTP Server is running on z/OS."})
	case "TYPE":
		writeLines(sess.conn, []string{"200 representation type is " + arg})
	case "SITE":
		writeLines(sess.conn, []string{"200 SITE command was accepted"})
	case "XSTA", "XSTAT":
		// Default: report a parseable FileType so dataset/spool flows can
		// query and restore it. Specific features should be scripted.
		writeLines(sess.conn, []string{"211-FileType SEQ (Sequential)", "211 *** end of status ***"})
	case "STAT":
		writeLines(sess.conn, []string{"211 mockzos status ok"})
	case "FEAT":
		writeLines(sess.conn, []string{"211-Extensions supported", "211 End"})
	case "REST":
		writeLines(sess.conn, []string{"350 restarting at the requested offset, send transfer command"})
	case "CWD":
		writeLines(sess.conn, []string{"250 directory changed"})
	case "NOOP":
		writeLines(sess.conn, []string{"200 command okay"})
	case "PASV":
		s.handlePasv(sess)
	case "LIST", "NLST", "RETR":
		s.handleDownload(sess, line, verb, arg)
	case "STOR", "STOU", "APPE":
		s.handleUpload(sess, arg)
	case "QUIT":
		writeLines(sess.conn, []string{"221 goodbye"})
		return true
	default:
		writeLines(sess.conn, []string{"200 command okay"})
	}
	return false
}

// handlePasv opens a fresh loopback data listener and advertises it.
func (s *Server) handlePasv(sess *session) {
	if sess.pasv != nil {
		_ = sess.pasv.Close()
		sess.pasv = nil
	}
	dl, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		writeLines(sess.conn, []string{"425 cannot open data connection"})
		return
	}
	sess.pasv = dl
	port := dl.Addr().(*net.TCPAddr).Port
	writeLines(sess.conn, []string{fmt.Sprintf("227 Entering Passive Mode (127,0,0,1,%d,%d)", port>>8, port&0xff)})
}

// handleDownload streams a registered payload over the passive data connection.
func (s *Server) handleDownload(sess *session, line, verb, arg string) {
	payload, ok := s.dataFor(line, verb)
	if !ok {
		payload = "" // empty listing is valid
	}
	dc := s.acceptData(sess)
	if dc == nil {
		writeLines(sess.conn, []string{"425 cannot open data connection"})
		return
	}
	writeLines(sess.conn, []string{"125 data connection already open; transfer starting"})
	_, _ = dc.Write([]byte(payload))
	_ = dc.Close()
	writeLines(sess.conn, []string{"250 transfer completed successfully"})
}

// handleUpload captures the payload the client sends over the data connection.
func (s *Server) handleUpload(sess *session, arg string) {
	dc := s.acceptData(sess)
	if dc == nil {
		writeLines(sess.conn, []string{"425 cannot open data connection"})
		return
	}
	writeLines(sess.conn, []string{"125 data connection already open; transfer starting"})
	buf := new(strings.Builder)
	_, _ = copyAll(buf, dc)
	_ = dc.Close()
	s.mu.Lock()
	s.stored[strings.ToUpper(strings.TrimSpace(arg))] = []byte(buf.String())
	s.mu.Unlock()
	writeLines(sess.conn, []string{"250 transfer completed successfully"})
}

// acceptData accepts the pending passive data connection.
func (s *Server) acceptData(sess *session) net.Conn {
	if sess.pasv == nil {
		return nil
	}
	defer func() {
		_ = sess.pasv.Close()
		sess.pasv = nil
	}()
	if tl, ok := sess.pasv.(*net.TCPListener); ok {
		_ = tl.SetDeadline(time.Now().Add(dataTimeout))
	}
	dc, err := sess.pasv.Accept()
	if err != nil {
		return nil
	}
	return dc
}

// splitCommand splits a request line into an uppercased verb and its argument.
func splitCommand(line string) (verb, arg string) {
	line = strings.TrimSpace(line)
	if i := strings.IndexByte(line, ' '); i >= 0 {
		return strings.ToUpper(line[:i]), strings.TrimSpace(line[i+1:])
	}
	return strings.ToUpper(line), ""
}

func writeLines(conn net.Conn, lines []string) {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l)
		b.WriteString("\r\n")
	}
	_, _ = conn.Write([]byte(b.String()))
}

// copyAll drains src into dst without importing io (keeps the helper explicit).
func copyAll(dst *strings.Builder, src net.Conn) (int, error) {
	buf := make([]byte, 4096)
	total := 0
	for {
		n, err := src.Read(buf)
		if n > 0 {
			dst.Write(buf[:n])
			total += n
		}
		if err != nil {
			return total, err
		}
	}
}
