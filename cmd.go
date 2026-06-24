// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"context"
	"errors"
	"fmt"
	"gopkg.in/ro-ag/zftp.v2/internal/log"
	"gopkg.in/ro-ag/zftp.v2/internal/utils"
	"net"
	"strings"
	"time"
)

// SendCommandWithContext sends a command to the FTP server and expects a specific
// return code. The context allows for cancellation or timeouts.
//
// The whole round-trip (write + reply) is serialized on the session mutex so the
// control stream is never read or written by two goroutines at once, making
// *FTPSession safe to share across goroutines.
func (s *FTPSession) SendCommandWithContext(ctx context.Context, expect ReturnCode, command string, a ...string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sendLocked(ctx, expect, command, a...)
}

// sendLocked performs a single control-connection round-trip. The caller must
// hold s.mu.
//
// Cancellation is honored by pushing the context deadline onto the connection and
// doing the write/read inline (no spawned goroutine). FTP has no in-band way to
// resynchronize a control stream once a reply is partially read, so any I/O-level
// failure — a deadline firing, EOF, or a reset — leaves the stream unrecoverable:
// the session is closed so later commands fail fast instead of reading a stale
// reply. A complete-but-unexpected reply (a *ReturnError) keeps the stream in
// sync and does not close the session.
func (s *FTPSession) sendLocked(ctx context.Context, expect ReturnCode, command string, a ...string) (string, error) {
	if s.isClosed.Load() {
		return "", fmt.Errorf("zftp: cannot send %s: session is closed", strings.ToUpper(strings.TrimSpace(command)))
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	conn, reader := s.conn, s.reader
	fullCommand := parseCommand(s.log, command, a...)

	if dl, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(dl); err != nil {
			return "", err
		}
		defer func() { _ = conn.SetDeadline(time.Time{}) }()
	}

	// log has already been printed in parseCommand
	if _, err := conn.Write(fullCommand); err != nil {
		s.log.Commandf("error %s", err)
		s.closeLocked()
		return "", fmt.Errorf("zftp: control connection write failed, session closed: %w", err)
	}

	msg, err := expect.check(reader, s.log)
	if err != nil {
		s.log.Serverf("error %s", err)
		var re *ReturnError
		if errors.As(err, &re) {
			// Reply read in full but with an unexpected (yet valid) FTP code; the
			// control stream is still in sync, so keep the session usable.
			return msg, err
		}
		// I/O-level failure: the control stream is desynchronized for good.
		s.closeLocked()
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", fmt.Errorf("zftp: command %s aborted (%w), session closed: %w",
				strings.ToUpper(strings.TrimSpace(command)), ctxErr, err)
		}
		return "", fmt.Errorf("zftp: control connection error, session closed: %w", err)
	}

	return msg, nil
}

// parseCommand parses a command and its arguments into a byte slice.
func parseCommand(lg *log.Logger, cmd string, a ...string) []byte {

	var (
		command = strings.TrimSpace(strings.ToUpper(cmd))
		args    = strings.TrimSpace(strings.Join(a, " "))
	)

	switch {
	case strings.HasPrefix(command, "PASS"):
		maskPassword := strings.Repeat("*", len(args))
		lg.Commandf("PASS %s", maskPassword)
	case len(a) > 0:
		lg.Commandf("%s %s", command, args)
	default:
		lg.Commandf("%s", command)
	}

	var fullCommand []byte
	if args == "" {
		// No arguments: emit "verb\r\n" with no trailing space (e.g. "NOOP\r\n").
		fullCommand = fmt.Appendf(nil, "%s\r\n", command)
	} else {
		fullCommand = fmt.Appendf(nil, "%s %s\r\n", command, args)
	}

	return fullCommand
}

// SendCommand sends a command to the FTP server and expects a specific return
// code. The whole round-trip is bounded by the session's reply timeout (see
// WithReplyTimeout, default 120s) so a server that accepts a command but never
// replies cannot hang the caller forever; on expiry the control stream is
// considered unrecoverable and the session is closed. Use SendCommandWithContext
// to supply a different deadline or cancellation.
func (s *FTPSession) SendCommand(expect ReturnCode, command string, a ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.dialCfg.replyTimeout())
	defer cancel()
	return s.SendCommandWithContext(ctx, expect, command, a...)
}

// CheckLast reads the server message buffer and validate the return code.
func (s *FTPSession) CheckLast(expect ReturnCode) (string, error) {
	return s.checkLast(expect)
}

// checkLast reads the terminal reply after a data transfer. It accepts the reply
// when its code is expect or any code in alsoAccept — a data transfer may finish
// with either 226 or 250 (see confirmData) — reports a *ReturnError on any other
// complete reply (the control stream stays in sync, so the session is kept), and
// closes the session on an I/O-level failure (the stream is then unrecoverable).
func (s *FTPSession) checkLast(expect ReturnCode, alsoAccept ...ReturnCode) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.dialCfg.replyTimeout())
	defer cancel()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isClosed.Load() {
		s.log.Warningf("<%s> session %s is closed", utils.Caller(), s.conn.RemoteAddr().String())
		// Never report a completion on a closed session: confirmData would otherwise
		// treat a transfer whose terminal reply was never read as successful.
		return "", net.ErrClosed
	}

	// Bound the reply read: z/OS sends the terminal 226/250 asynchronously to the
	// data-connection close and it can be lost, which would otherwise hang here.
	if dl, ok := ctx.Deadline(); ok {
		if err := s.conn.SetDeadline(dl); err != nil {
			return "", err
		}
		defer func() { _ = s.conn.SetDeadline(time.Time{}) }()
	}

	s.lastMessage.Reset()

	msg, err := expect.check(s.reader, s.log)

	s.lastMessage.WriteString(msg)

	if err != nil {
		var re *ReturnError
		if errors.As(err, &re) {
			// A complete, in-sync reply whose code is not expect. Accept it when it is
			// one of the alternate completion codes (e.g. 226 vs 250); the stream is
			// still synchronized, so on a genuine mismatch the session is kept open and
			// the error surfaced.
			for _, code := range alsoAccept {
				if re.rc == int(code) {
					return msg, nil
				}
			}
			s.log.Serverf("[res|error] %s", err)
			// A post-transfer reply whose code is not a completion code (e.g. 426
			// "transfer aborted", which z/OS may follow with a second reply such as
			// 226 acknowledging an ABOR) means the transfer did not end cleanly. Such
			// replies can be multi-part, and FTP has no in-band way to resynchronize a
			// partially-drained control stream, so close the session rather than leave
			// a trailing reply buffered to desync ("shift") the next command.
			s.closeLocked()
			return "", err
		}
		s.log.Serverf("[res|error] %s", err)
		// I/O-level failure on the post-transfer reply read: like sendLocked, the
		// control stream is desynchronized for good, so close the session.
		s.closeLocked()
		return "", err
	}

	return msg, nil
}

// System returns the operating-system type reported by the FTP server.
//
// When the value is already known — it is cached during Login — it is returned
// with a nil error and no network round-trip. Otherwise a SYST command is issued
// and its reply (or the resulting error) is returned. A control-connection or
// protocol failure is surfaced as an error, never a panic.
func (s *FTPSession) System() (string, error) {
	s.mu.Lock()
	sys := s.system
	s.mu.Unlock()
	if sys != "" {
		return sys, nil
	}

	return s.SendCommand(CodeSysType, "SYST")
}

// CWD changes the current working directory to the specified path.
func (s *FTPSession) CWD(expression string) (string, error) {
	return s.SendCommand(CodeFileActionOK, "CWD", expression)
}
