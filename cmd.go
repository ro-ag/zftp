package zftp

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ro-ag/zftp.v0/internal/utils"
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
			log.Debugf("[cmd|error] %s", err)
			errChan <- err
			return
		}

		msg, err := expect.check(s.reader)
		if err != nil {
			log.Debugf("[res|error] %s", err)
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
		log.Debugf("[cmd] PASS %s", maskPassword)
	case len(a) > 0:
		log.Debugf("[cmd] %s %s", command, args)
	default:
		log.Debugf("[cmd] %s", command)
	}

	fullCommand := []byte(fmt.Sprintf("%s %s\r\n", command, args))

	return fullCommand
}

// SendCommand sends a command to the FTP server and expects a specific return code. It uses a default context.
func (s *FTPSession) SendCommand(expect ReturnCode, command string, a ...string) (string, error) {
	return s.SendCommandWithContext(context.Background(), expect, command, a...)
}

// Site sends the SITE command to the FTP server.
func (s *FTPSession) Site(subCommand string, a ...string) (string, error) {
	args := strings.Join(a, " ")
	subCommand = strings.TrimSpace(strings.ToUpper(subCommand))
	subCommandWithArgs := fmt.Sprintf("%s %s", subCommand, args)
	str, err := s.SendCommand(CodeCmdOK, "SITE", subCommandWithArgs)
	lines := strings.Split(str, "\n")
	switch {
	case err != nil:
		return "", err
	case strings.Contains(str, "Unrecognized parameter"):
		return "", fmt.Errorf("error : '%s', %s", subCommandWithArgs, lines[0])
	case strings.Contains(str, "Parameter ignored"):
		return "", fmt.Errorf("error : '%s', %s", subCommandWithArgs, lines[0])
	default:
		return str, nil
	}
}

func (s *FTPSession) checkLast(expect ReturnCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isClosed.Load() {
		log.Warningf("<%s> session %s is closed", utils.Caller(), s.conn.RemoteAddr().String())
		return nil
	}

	_, err := expect.check(s.reader)
	if err != nil {
		log.Debugf("[res|error] %s", err)
		return err
	}

	return nil
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
