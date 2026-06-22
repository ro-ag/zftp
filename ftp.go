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
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
)

// FTPSession represents an FTP session
type FTPSession struct {
	conn        net.Conn
	system      string
	user        string
	currType    TransferType
	jobPrefix   *regexp.Regexp
	isClosed    atomic.Bool
	reader      *bufio.Reader
	lastMessage strings.Builder
	dataConns   sync.Map
	tlsConfig   *tls.Config
	dialCfg     dialOptions
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

	msg, err := CodeSvcReadySoon.check(session.reader)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	log.Debug(utils.WrapText(msg))

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
		reader:    bufio.NewReader(conn),
		dialCfg:   cfg,
		jobPrefix: regexp.MustCompile(`(JOB\d{5})`),
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
			log.Errorf("error closing FTP session: %s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
}

// SetVerbose sets the verbose flag
func (s *FTPSession) SetVerbose(level LogLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.SetLevel(log.Level(level))
}

// Conn exposes the underlying network connection used by the session.
// This allows the caller to apply custom options such as TCP keep alive.
func (s *FTPSession) Conn() net.Conn {
	return s.conn
}

// AuthTLS sends the AUTH TLS command to the FTP server and sets up the TLS connection
func (s *FTPSession) AuthTLS(tlsConfig *tls.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.SendCommand(CodeSecurityOk, "AUTH", "TLS")
	if err != nil {
		return err
	}

	s.tlsConfig = tlsConfig

	s.conn = tls.Client(s.conn, tlsConfig)

	s.reader = bufio.NewReader(s.conn)

	// Protection Buffer Size
	_, err = s.SendCommand(CodeCmdOK, "PBSZ", "0")
	if err != nil {
		return err
	}

	// data Channel Protection Level
	_, err = s.SendCommand(CodeCmdOK, "PROT", "P")
	if err != nil {
		return err
	}

	return nil
}

// Close closes all connections to the FTP server
func (s *FTPSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Close all data connections
	s.dataConns.Range(func(name, conn any) bool {
		child := conn.(*childConnection)
		log.Debugf("closing child net connection %s", child.RemoteAddr())
		err := child.Close()
		if err != nil {
			log.Warningf("Error closing child net connection %s: %s", child.RemoteAddr(), err)
		}
		return true
	})

	// Send QUIT command and close main connection

	log.Debugf("closing session connection %s", s.conn.RemoteAddr())

	err := s.conn.Close()
	if err != nil {
		log.Warningf("Error closing session connection: %s", err)
		return err
	}
	s.isClosed.Store(true)
	return nil
}

// Login sends the USER and PASS commands to the FTP server
func (s *FTPSession) Login(user, pass string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.user = strings.ToUpper(user)

	_, err := s.SendCommand(CodeNeedPwd, "USER", user)
	if err != nil {
		return err
	}

	_, err = s.SendCommand(CodeLoggedInProceed, "PASS", pass)
	if err != nil {
		return err
	}
	// set passive mode
	_, err = s.SendCommand(CodeEnteringPassiveMode, "PASV")
	if err != nil {
		return err
	}

	/* Set default type to Image or Binary */
	err = s.SetType(TypeImage)
	if err != nil {
		return err
	}

	/* Indicate mainframe set End of line default per system */
	err = s.SetStatusOf().SBSendEol(eol.System)
	if err != nil {
		return err
	}

	/* Indicate mainframe set End of line default per system */
	err = s.SetStatusOf().MBSendEol(eol.System)
	if err != nil {
		return err
	}

	/* Check */
	syt, err := s.SendCommand(CodeSysType, "SYST")
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
	return s.user
}
