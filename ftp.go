package zftp

import (
	"bufio"
	"crypto/tls"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ro-ag/zftp.v0/eol"
	"gopkg.in/ro-ag/zftp.v0/internal/utils"
	"net"
	"os"
	"os/signal"
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
	isClosed    atomic.Bool
	reader      *bufio.Reader
	lastMessage strings.Builder
	dataConns   sync.Map
	verbose     bool
	tlsConfig   *tls.Config
	mu          sync.Mutex
}

// Open opens a Net connection to the FTP server and returns an FTPSession
func Open(server string) (*FTPSession, error) {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		return nil, err
	}

	session := &FTPSession{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}

	msg, err := CodeSvcReadySoon.check(session.reader)
	if err != nil {
		return nil, err
	}
	log.Info(utils.WrapText(msg))
	// Setup signal handler
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		if err = session.Close(); err != nil {
			log.Errorf("error closing FTP session: %s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()

	return session, nil
}

// SetVerbose sets the verbose flag
func (s *FTPSession) SetVerbose(v bool) {
	s.mu.Lock()
	s.verbose = v
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:      true,
		FullTimestamp:    true,
		DisableTimestamp: true,
		//	PadLevelText:     true,
	})
	s.mu.Unlock()
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
		log.Debugf("[***] closing child net connection %s", child.RemoteAddr())
		err := child.Close()
		if err != nil {
			log.Warningf("Error closing child net connection %s: %s", child.RemoteAddr(), err)
		}
		return true
	})

	// Send QUIT command and close main connection

	log.Debugf("[***] closing session connection %s", s.conn.RemoteAddr())

	err := s.conn.Close()
	if err != nil {
		log.Printf("Error closing session connection: %s", err)
		return err
	}
	s.isClosed.Store(true)
	return nil
}

// Login sends the USER and PASS commands to the FTP server
func (s *FTPSession) Login(user, pass string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if user == "" {
		user = os.Getenv("FTP_USER")
	}
	if pass == "" {
		pass = os.Getenv("FTP_PASS")
	}

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
