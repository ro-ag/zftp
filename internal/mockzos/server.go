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
	"crypto/tls"
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

	mu              sync.Mutex
	lineScripts     map[string][]string // full command line (upper) -> raw reply lines
	verbScripts     map[string][]string // verb (upper) -> raw reply lines
	dataByLine      map[string]string   // full command line (upper) -> payload to send
	dataByVerb      map[string]string   // verb (upper) -> payload to send
	stored          map[string][]byte   // STOR arg (upper) -> captured payload
	withheld        map[string]bool     // full line or verb (upper) -> swallow without replying
	hangup          map[string]bool     // full line or verb (upper) -> drop the control conn without replying
	dropAfterData   map[string]bool     // download verb (upper) -> drop control after data, before the closing reply
	truncate        map[string]bool     // download verb (upper) -> RST the data conn instead of a clean close
	hangData        map[string]bool     // download verb (upper) -> hold the data conn open after sending payload
	withholdReply   map[string]bool     // download verb (upper) -> deliver data + clean close, but send no closing reply
	completionReply map[string][]string // transfer verb (upper) -> override the closing reply (default "250 ...")
	tlsConfig       *tls.Config         // when set, AUTH TLS upgrades the control connection
	received        []string            // every command line received, in order
}

// New starts a Server on 127.0.0.1:0 and registers cleanup with the test.
func New(tb testing.TB) *Server {
	tb.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatalf("mockzos listen: %v", err)
	}
	s := &Server{
		tb:              tb,
		ln:              ln,
		lineScripts:     map[string][]string{},
		verbScripts:     map[string][]string{},
		dataByLine:      map[string]string{},
		dataByVerb:      map[string]string{},
		stored:          map[string][]byte{},
		withheld:        map[string]bool{},
		hangup:          map[string]bool{},
		dropAfterData:   map[string]bool{},
		truncate:        map[string]bool{},
		hangData:        map[string]bool{},
		withholdReply:   map[string]bool{},
		completionReply: map[string][]string{},
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
		s.wg.Go(func() {
			s.handle(conn)
		})
	}
}

// session holds per-connection state: the (possibly TLS-upgraded) control
// connection, its buffered reader, and the pending passive data listener.
type session struct {
	conn net.Conn
	r    *bufio.Reader
	pasv net.Listener
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	sess := &session{conn: conn, r: bufio.NewReader(conn)}
	defer func() {
		if sess.pasv != nil {
			_ = sess.pasv.Close()
		}
	}()

	writeLines(conn, []string{"220 mockzos FTP service ready"})

	for {
		// Read from sess.r, not a captured reader: AUTH TLS swaps both the
		// connection and its reader, so subsequent commands must be read through
		// the upgraded reader.
		line, err := sess.r.ReadString('\n')
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
	// A withheld command is consumed without any reply, modeling a hung control
	// connection so the client's timeout/cancel paths can be exercised. The
	// connection stays open; the client is expected to give up and close it.
	if s.isWithheld(line, verb) {
		return false
	}
	// A hung-up command drops the control connection with no reply, modeling a
	// peer that closes the control stream (EOF) instead of answering.
	if s.isHangup(line, verb) {
		return true
	}
	// Explicit scripted replies take precedence over defaults.
	if reply, ok := s.scriptFor(line, verb); ok {
		writeLines(sess.conn, reply)
		return verb == "QUIT"
	}

	switch verb {
	case "AUTH":
		s.handleAuth(sess, arg)
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
		s.handleUpload(sess, verb, arg)
	case "QUIT":
		writeLines(sess.conn, []string{"221 goodbye"})
		return true
	default:
		writeLines(sess.conn, []string{"200 command okay"})
	}
	return false
}

// handleAuth answers AUTH TLS when TLS has been enabled via EnableTLS: it replies
// 234 and upgrades the control connection to a TLS server session, swapping the
// connection and its reader so every later command on this session runs
// encrypted. AUTH with any other mechanism, or when TLS is not enabled, is
// rejected with 504.
func (s *Server) handleAuth(sess *session, arg string) {
	s.mu.Lock()
	cfg := s.tlsConfig
	s.mu.Unlock()
	if cfg == nil || !strings.EqualFold(strings.TrimSpace(arg), "TLS") {
		writeLines(sess.conn, []string{"504 security mechanism not supported"})
		return
	}
	writeLines(sess.conn, []string{"234 security environment established, ready for TLS negotiation"})
	tconn := tls.Server(sess.conn, cfg)
	if err := tconn.Handshake(); err != nil {
		s.tb.Logf("mockzos: TLS handshake failed: %v", err)
		_ = sess.conn.Close()
		return
	}
	sess.conn = tconn
	sess.r = bufio.NewReader(tconn)
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

	// HangData: hold the data connection open after sending the payload so the
	// client's scan blocks; block here until the client tears the connection down
	// (e.g. via a concurrent Close).
	if s.isHangData(verb) {
		var sink strings.Builder
		_, _ = copyAll(&sink, dc)
		return
	}

	// TruncateData: abort the data connection with a RST (SO_LINGER 0) instead of a
	// clean FIN, modeling a failed/aborted z/OS transfer. The control reply still
	// says 250, so only the data-stream error should fail the operation.
	if s.isTruncateData(verb) {
		if tcp, ok := dc.(*net.TCPConn); ok {
			_ = tcp.SetLinger(0)
		}
		_ = dc.Close()
		writeLines(sess.conn, []string{"250 transfer completed successfully"})
		return
	}

	_ = dc.Close()
	// DropControlAfterData: drop the control connection after the data has been
	// delivered but before the closing reply, so the reply read hits EOF.
	if s.isDropAfterData(verb) {
		_ = sess.conn.Close()
		return
	}
	// WithholdReplyAfterData: deliver and cleanly close the data, but never send the
	// closing reply, leaving the control connection open so the reply read blocks
	// and must time out.
	if s.isWithholdReplyAfterData(verb) {
		return
	}
	writeLines(sess.conn, s.completionReplyFor(verb))
}

// handleUpload captures the payload the client sends over the data connection.
func (s *Server) handleUpload(sess *session, verb, arg string) {
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
	writeLines(sess.conn, s.completionReplyFor(verb))
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
	if before, after, ok := strings.Cut(line, " "); ok {
		return strings.ToUpper(before), strings.TrimSpace(after)
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
