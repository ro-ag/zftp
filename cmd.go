package zftp

import (
	"context"
	"fmt"
	"gopkg.in/ro-ag/zftp.v1/internal/log"
	"gopkg.in/ro-ag/zftp.v1/internal/utils"
	"strings"
)

// SendCommandWithContext sends a command to the FTP server and expects a specific return code.
// The context allows for cancellation or timeouts.
func (s *FTPSession) SendCommandWithContext(ctx context.Context, expect ReturnCode, command string, a ...string) (string, error) {

	var (
		errChan     = make(chan error, 1)
		respChan    = make(chan string, 1)
		fullCommand = parseCommand(command, a...)
	)

	go func() {
		// log has already been printed in parseCommand
		_, err := s.conn.Write(fullCommand)
		if err != nil {
			log.Commandf("error %s", err)
			errChan <- err
			return
		}

		msg, err := expect.check(s.reader)
		if err != nil {
			log.Serverf("error %s", err)
			errChan <- err
			return
		}

		respChan <- msg
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-errChan:
		return "", err
	case resp := <-respChan:
		return resp, nil
	}
}

// parseCommand parses a command and its arguments into a byte slice.
func parseCommand(cmd string, a ...string) []byte {

	var (
		command = strings.TrimSpace(strings.ToUpper(cmd))
		args    = strings.TrimSpace(strings.Join(a, " "))
	)

	switch {
	case strings.HasPrefix(command, "PASS"):
		maskPassword := strings.Repeat("*", len(args))
		log.Commandf("PASS %s", maskPassword)
	case len(a) > 0:
		log.Commandf("%s %s", command, args)
	default:
		log.Commandf("%s", command)
	}

	fullCommand := []byte(fmt.Sprintf("%s %s\r\n", command, args))

	return fullCommand
}

// SendCommand sends a command to the FTP server and expects a specific return code. It uses a default context.
func (s *FTPSession) SendCommand(expect ReturnCode, command string, a ...string) (string, error) {
	return s.SendCommandWithContext(context.Background(), expect, command, a...)
}

// CheckLast reads the server message buffer and validate the return code.
func (s *FTPSession) CheckLast(expect ReturnCode) (string, error) {
	return s.checkLast(expect)
}

func (s *FTPSession) checkLast(expect ReturnCode) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isClosed.Load() {
		log.Warningf("<%s> session %s is closed", utils.Caller(), s.conn.RemoteAddr().String())
		return "", nil
	}

	s.lastMessage.Reset()

	msg, err := expect.check(s.reader)

	s.lastMessage.WriteString(msg)

	if err != nil {
		log.Serverf("[res|error] %s", err)
		return "", err
	}

	return msg, nil
}

// System get the system type of the FTP server. will panic
func (s *FTPSession) System() string {
	if s.system != "" {
		return s.system
	}

	system, err := s.SendCommand(CodeSysType, "SYST")
	if err != nil {
		panic(err)
	}
	return system
}

// CWD changes the current working directory to the specified path.
func (s *FTPSession) CWD(expression string) (string, error) {
	return s.SendCommand(CodeFileActionOK, "CWD", expression)
}
