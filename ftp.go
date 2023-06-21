package zftp

import (
	"bufio"
	"crypto/tls"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ro-ag/zftp.v0/internal/utils"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
)

type FTPSession struct {
	conn        net.Conn
	system      string
	currType    TransferType
	isClosed    atomic.Bool
	r           *bufio.Reader
	lastMessage strings.Builder
	dataConns   sync.Map
	verbose     bool
	tlsConfig   *tls.Config
	mu          sync.Mutex
}

// Open opens a TLS connection to the FTP server and returns an FTPSession
func Open(server string) (*FTPSession, error) {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		return nil, err
	}

	session := &FTPSession{
		conn: conn,
		r:    bufio.NewReader(conn),
	}

	msg, err := CodeSvcReadySoon.CheckCode(session.r)
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

	syt, err := session.SendCommand(CodeSysType, "SYST")
	if err != nil {
		return nil, err
	}

	if !strings.Contains(syt, "MVS") {
		return nil, fmt.Errorf("unsupported system type: %s", syt)
	}

	session.system = "MVS"

	return session, nil
}

func (s *FTPSession) SetVerbose(v bool) {
	s.mu.Lock()
	s.verbose = v
	s.mu.Unlock()
}

// AuthTLS sends the AUTH TLS command to the FTP server and sets up the TLS connection
func (s *FTPSession) AuthTLS(tlsConfig *tls.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.SendCommand(CodeSecurityExchangeOK, "AUTH", "TLS")
	if err != nil {
		return err
	}

	s.tlsConfig = tlsConfig

	s.conn = tls.Client(s.conn, tlsConfig)

	s.r = bufio.NewReader(s.conn)

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
	err = s.SetRetrieveEOL(EolSystem)
	if err != nil {
		return err
	}

	/* Indicate mainframe set End of line default per system */
	err = s.SetRetrieveWideCharEOL(EolSystem)
	if err != nil {
		return err
	}

	return nil
}

func drainBuffer(r *bufio.Reader) {
	// Read until there's nothing left to read
	for {
		_, err := r.Peek(1)
		if err != nil {
			// If Peek returns an error (including io.EOF), we stop
			break
		}
		r.ReadByte()
	}
}
