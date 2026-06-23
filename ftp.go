// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"gopkg.in/ro-ag/zftp.v2/eol"
	"gopkg.in/ro-ag/zftp.v2/internal/log"
	"gopkg.in/ro-ag/zftp.v2/internal/utils"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// FTPSession represents an FTP session
type FTPSession struct {
	conn        net.Conn
	rawConn     net.Conn // underlying socket, never swapped on TLS upgrade; lets Close interrupt a blocked control read
	system      string
	user        string
	currType    atomic.Uint32 // current TransferType; atomic so transfers can read it lock-free
	jobPrefix   *regexp.Regexp
	isClosed    atomic.Bool
	reader      *bufio.Reader
	lastMessage strings.Builder
	dataConns   sync.Map
	tlsConfig   *tls.Config
	dialCfg     dialOptions
	log         *log.Logger
	mu          sync.Mutex
}

// Open opens a network connection to the FTP server, reads the greeting, and
// returns an FTPSession. The control connection is obtained through the
// configured Dialer (see WithDialer); by default a standard *net.Dialer is used.
func Open(server string, opts ...Option) (*FTPSession, error) {
	var cfg dialOptions
	cfg.apply(opts)

	conn, err := dialControl(server, cfg)
	if err != nil {
		return nil, err
	}

	session := newSession(conn, cfg)

	msg, err := CodeSvcReadySoon.check(session.reader, session.log)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	session.log.Debug(utils.WrapText(msg))

	if cfg.signalHandler {
		session.installSignalHandler()
	}

	return session, nil
}

// dialControl establishes the control connection using the configured dialer,
// falling back to a standard *net.Dialer with the configured timeout/keep-alive.
func dialControl(server string, cfg dialOptions) (net.Conn, error) {
	if cfg.dialer != nil {
		return cfg.dialer.DialContext(context.Background(), "tcp", server)
	}

	dialer := net.Dialer{Timeout: cfg.DialTimeout}
	if cfg.KeepAlivePeriod > 0 {
		dialer.KeepAlive = cfg.KeepAlivePeriod
	}

	conn, err := dialer.Dial("tcp", server)
	if err != nil {
		return nil, err
	}

	if tcp, ok := conn.(*net.TCPConn); ok && cfg.KeepAlivePeriod > 0 {
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(cfg.KeepAlivePeriod)
	}

	return conn, nil
}

// newSession wraps an established control connection in an FTPSession. It is the
// unexported construction seam shared by Open and by in-process tests.
func newSession(conn net.Conn, cfg dialOptions) *FTPSession {
	return &FTPSession{
		conn:      conn,
		rawConn:   conn,
		reader:    bufio.NewReader(conn),
		dialCfg:   cfg,
		jobPrefix: regexp.MustCompile(`(JOB\d{5})`),
		log:       log.New(cfg.logger, log.None),
	}
}

// installSignalHandler tears the session down on SIGINT/SIGTERM and exits.
// Only installed when WithSignalHandler is set.
func (s *FTPSession) installSignalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		if err := s.Close(); err != nil {
			s.log.Errorf("error closing FTP session: %s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
}

// SetVerbose selects which trace categories this session emits. The level is a
// bitmask of the Log* constants (or NoLog/LogAll) and is independent of any
// injected logger's own level filter.
func (s *FTPSession) SetVerbose(level LogLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.log.SetLevel(log.Level(level))
}

// SetLogger swaps the destination logger for this session's logs at runtime.
// A nil l reverts to slog.Default(). See WithLogger to set it at Open time.
func (s *FTPSession) SetLogger(l *slog.Logger) {
	s.log.SetSlog(l)
}

// SetKeepAlive enables TCP keep-alive on the underlying control socket with the
// given idle period, or disables keep-alive when d <= 0. It returns an error if
// the underlying connection is not a *net.TCPConn (for example, a custom dialer
// supplied a different net.Conn implementation).
//
// Read/write deadlines and Read/Write/Close are deliberately not exposed: the
// command layer owns deadlines and connection lifecycle, and handing out the raw
// socket would let callers corrupt the control stream or bypass session
// bookkeeping. Use RemoteAddr/LocalAddr for the peer/local addresses.
func (s *FTPSession) SetKeepAlive(d time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tcp, ok := s.rawConn.(*net.TCPConn)
	if !ok {
		return fmt.Errorf("zftp: SetKeepAlive: underlying connection is not a *net.TCPConn (%T)", s.rawConn)
	}
	if d <= 0 {
		return tcp.SetKeepAlive(false)
	}
	if err := tcp.SetKeepAlive(true); err != nil {
		return err
	}
	return tcp.SetKeepAlivePeriod(d)
}

// RemoteAddr returns the remote network address of the underlying control
// connection (the FTP server), or nil if it is unavailable.
func (s *FTPSession) RemoteAddr() net.Addr {
	return s.rawConn.RemoteAddr()
}

// LocalAddr returns the local network address of the underlying control
// connection, or nil if it is unavailable.
func (s *FTPSession) LocalAddr() net.Addr {
	return s.rawConn.LocalAddr()
}

// AuthTLS sends the AUTH TLS command to the FTP server and sets up the TLS connection
func (s *FTPSession) AuthTLS(tlsConfig *tls.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Already holding s.mu: use sendLocked to avoid re-entrant deadlock, and keep
	// the AUTH negotiation and the conn/reader swap atomic against other commands.
	_, err := s.sendLocked(context.Background(), CodeSecurityOk, "AUTH", "TLS")
	if err != nil {
		return err
	}

	s.tlsConfig = tlsConfig

	s.conn = tls.Client(s.conn, tlsConfig)

	s.reader = bufio.NewReader(s.conn)

	// Protection Buffer Size
	_, err = s.sendLocked(context.Background(), CodeCmdOK, "PBSZ", "0")
	if err != nil {
		return err
	}

	// data Channel Protection Level
	_, err = s.sendLocked(context.Background(), CodeCmdOK, "PROT", "P")
	if err != nil {
		return err
	}

	return nil
}

// Close closes all connections to the FTP server. It is idempotent and safe to
// call concurrently with in-flight commands: Close and the command path both take
// the session mutex, so a command in progress finishes before the connection is
// torn down (and a command that starts after Close fails cleanly).
func (s *FTPSession) Close() error {
	// Interrupt any control read currently blocked under s.mu so its goroutine
	// returns and releases the lock; otherwise Close would block forever behind a
	// command stalled on a silent peer. rawConn is the underlying socket (never
	// swapped on a TLS upgrade), so this interrupts plaintext and TLS reads alike.
	// It is safe to touch without the lock: rawConn is set once at construction.
	_ = s.rawConn.SetDeadline(time.Now())

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closeLocked()
}

// closeLocked tears down the control connection and every data connection. It is
// idempotent. The caller must hold s.mu. It is also invoked from the send path
// when an I/O error leaves the control stream unrecoverable.
func (s *FTPSession) closeLocked() error {
	if s.isClosed.Swap(true) {
		return nil
	}

	// Close all data connections.
	s.dataConns.Range(func(name, conn any) bool {
		child := conn.(*childConnection)
		s.log.Debugf("closing child net connection %s", child.RemoteAddr())
		if err := child.Close(); err != nil {
			s.log.Warningf("Error closing child net connection %s: %s", child.RemoteAddr(), err)
		}
		return true
	})

	s.log.Debugf("closing session connection %s", s.conn.RemoteAddr())
	if err := s.conn.Close(); err != nil {
		s.log.Warningf("Error closing session connection: %s", err)
		return err
	}
	return nil
}

// IsClosed reports whether the session has been closed — either explicitly via
// Close or implicitly after an unrecoverable control-connection error (for
// example a context timeout that aborts an in-flight command). A closed session
// rejects further commands.
func (s *FTPSession) IsClosed() bool {
	return s.isClosed.Load()
}

// Login sends the USER and PASS commands to the FTP server
func (s *FTPSession) Login(user, pass string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.user = strings.ToUpper(user)

	// The whole login handshake runs under s.mu, so every step uses the locked
	// helpers (sendLocked / setTypeLocked / setStatusOfLocked) to avoid the
	// re-entrant deadlock a sync.Mutex would otherwise cause.
	_, err := s.sendLocked(context.Background(), CodeNeedPwd, "USER", user)
	if err != nil {
		return err
	}

	_, err = s.sendLocked(context.Background(), CodeLoggedInProceed, "PASS", pass)
	if err != nil {
		return err
	}
	// set passive mode
	_, err = s.sendLocked(context.Background(), CodeEnteringPassiveMode, "PASV")
	if err != nil {
		return err
	}

	/* Set default type to Image or Binary */
	err = s.setTypeLocked(TypeImage)
	if err != nil {
		return err
	}

	/* Indicate mainframe set End of line default per system */
	err = s.setStatusOfLocked().SBSendEol(eol.System)
	if err != nil {
		return err
	}

	/* Indicate mainframe set End of line default per system */
	err = s.setStatusOfLocked().MBSendEol(eol.System)
	if err != nil {
		return err
	}

	/* Check */
	syt, err := s.sendLocked(context.Background(), CodeSysType, "SYST")
	if err != nil {
		return err
	}

	if !strings.Contains(syt, "MVS") {
		return fmt.Errorf("unsupported system type: %s", syt)
	}

	s.system = "MVS"

	return nil
}

// GetUser returns the current username
func (s *FTPSession) GetUser() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.user
}
