// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"bufio"
	"fmt"
	"gopkg.in/ro-ag/zftp.v2/internal/log"
	"io"
	"strconv"
	"strings"
)

//go:generate stringer -type=ReturnCode

// ReturnCode is an FTP return code
type ReturnCode int

// FTP reply codes, per RFC 959 and the z/OS FTP dialect. They name the replies
// the client expects from the server and matches each command's result against.
const (
	CodeListOK                 ReturnCode = 125
	CodeFileStatusOK           ReturnCode = 150
	CodeDirStatusOK            ReturnCode = 151
	CodeCmdOK                  ReturnCode = 200
	CodeCmdNotImplementedSuper ReturnCode = 202
	CodeSysStatus              ReturnCode = 211
	CodeDirStatus              ReturnCode = 212
	CodeFileStatus             ReturnCode = 213
	CodeHelpMsg                ReturnCode = 214
	CodeSysType                ReturnCode = 215
	CodeSvcReadySoon           ReturnCode = 220
	CodeSvcClosingControlConn  ReturnCode = 221
	CodeDataConnOpen           ReturnCode = 225
	CodeClosingDataConn        ReturnCode = 226
	CodeEnteringPassiveMode    ReturnCode = 227
	CodeLoggedInProceed        ReturnCode = 230
	CodeSecurityOk             ReturnCode = 234
	CodeFileActionOK           ReturnCode = 250
	CodeDirCreated             ReturnCode = 257
	CodeNeedPwd                ReturnCode = 331
	CodeNeedAcctForLogin       ReturnCode = 332
	CodeSecurityExchangeOK     ReturnCode = 334
	CodeNeedInfo               ReturnCode = 350
	CodeSvcNotAvailable        ReturnCode = 421
	CodeCantOpenDataConn       ReturnCode = 425
	CodeConnClosed             ReturnCode = 426
	CodeFileActionNotTaken     ReturnCode = 450
	CodeLocalError             ReturnCode = 451
	CodeInsufficientStorage    ReturnCode = 452
	CodeCmdNotRecognized       ReturnCode = 500
	CodeArgsError              ReturnCode = 501
	CodeCmdNotImplemented      ReturnCode = 502
	CodeBadCmdSequence         ReturnCode = 503
	CodeCmdNotImplementedParam ReturnCode = 504
	CodeUserNotLogged          ReturnCode = 530
	CodeFileActionNotTakenPerm ReturnCode = 550
	CodePageTypeUnknown        ReturnCode = 551
	CodeExceededStorageAlloc   ReturnCode = 552
	CodeBadFileName            ReturnCode = 553
)

// ReturnError reports that the server answered with an FTP reply whose code was
// not the one the command expected. It carries the code received (rc), the code
// wanted (wantRc), the reply text, and optionally an underlying transport cause.
type ReturnError struct {
	rc      int
	message string
	wantRc  int
	cause   error // optional underlying transport cause, exposed via Unwrap
}

// ReturnCode returns the FTP return code
func (e *ReturnError) ReturnCode() ReturnCode {
	return ReturnCode(e.rc)
}

// Error returns the error message
func (e *ReturnError) Error() string {
	return fmt.Sprintf("FTP response code: got %d, want %d, message: %s", e.rc, e.wantRc, e.message)
}

// Is reports whether target is a *ReturnError carrying the same FTP return code.
// It lets callers match a result by code with errors.Is — for example
// errors.Is(err, zftp.CodeError(zftp.CodeFileActionNotTakenPerm)) — without
// unpacking the error or comparing ReturnCode() by hand. The expected code and
// message are intentionally ignored, so any failure carrying that code matches.
// Non-*ReturnError targets return false so errors.Is continues on to Unwrap.
func (e *ReturnError) Is(target error) bool {
	t, ok := target.(*ReturnError)
	return ok && t.rc == e.rc
}

// Unwrap returns the underlying transport cause attached to this error, or nil
// for a pure protocol error (a well-formed but unexpected FTP reply). When a
// cause is present — e.g. an io.ErrUnexpectedEOF for a reply cut short by a
// dropped control stream — errors.Is/errors.As can traverse to it.
func (e *ReturnError) Unwrap() error {
	return e.cause
}

// Compile-time checks that *ReturnError implements error and the optional Is /
// Unwrap interfaces the errors package looks for during errors.Is/errors.As.
var (
	_ error                       = (*ReturnError)(nil)
	_ interface{ Is(error) bool } = (*ReturnError)(nil)
	_ interface{ Unwrap() error } = (*ReturnError)(nil)
)

// CodeError returns an error usable only as an errors.Is target: it matches any
// *ReturnError carrying the given FTP return code. External callers cannot build
// a *ReturnError directly (its fields are unexported), so this is the supported
// way to test a result by code:
//
//	if errors.Is(err, zftp.CodeError(zftp.CodeFileActionNotTakenPerm)) {
//		// the server rejected the request with 550
//	}
func CodeError(rc ReturnCode) error {
	return &ReturnError{rc: int(rc), wantRc: int(rc)}
}

// check reads a (possibly multiline) FTP reply and returns its message.
//
// Per RFC 959 §4.2 a reply is one or more lines; the first parseable line sets
// the reply's code, and the reply terminates only on a line that repeats that
// exact OPENING code followed by a space. Intermediate continuation lines may
// contain anything — including text that itself begins with "NNN " for some
// other code (e.g. a z/OS message quoting "550 dataset..." inside a 211 block),
// which must NOT be mistaken for the terminator. Anchoring the terminator to the
// opening code keeps such replies whole and the control stream in sync for the
// next command. Every line is appended (including lines shorter than 4 bytes) so
// no reply text is lost.
func (code ReturnCode) check(reader *bufio.Reader, lg *log.Logger) (msg string, err error) {

	var (
		response    strings.Builder
		openingCode = 0
		haveOpening = false
	)

	for {
		line, isPrefix, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				// Reaching EOF here means the stream ended before a terminator line
				// (a real terminator breaks out of the loop below, so it never
				// reaches this read). Whether or not an opening line was seen, the
				// reply is truncated and the control stream is dead: surface it as a
				// plain io.ErrUnexpectedEOF so callers close the desynchronized
				// session instead of acting on a partial reply or mistaking it for a
				// ReturnError. This holds even when the opening code equals the
				// expected code, where the post-loop check would otherwise pass.
				return response.String(), io.ErrUnexpectedEOF
			}
			return "", err
		}

		lg.Serverf("%s", line)

		// Parse the leading 3-digit code when the line is long enough to carry
		// one. The first line that parses fixes the reply's opening code.
		lineCode := 0
		haveLineCode := false
		if len(line) >= 4 {
			if c, atoiErr := strconv.Atoi(string(line[:3])); atoiErr != nil {
				lg.Errorf("converting response code to integer: %s", atoiErr)
			} else {
				lineCode = c
				haveLineCode = true
				if !haveOpening {
					openingCode = lineCode
					haveOpening = true
				}
			}
		}

		// Append every line so no reply text is lost. Strip the redundant "NNN "
		// prefix only on lines that carry the expected code; keep continuation or
		// error lines whole so their codes stay visible in the message.
		if haveLineCode && lineCode == int(code) {
			response.Write(line[4:])
		} else {
			response.Write(line)
		}

		// The reply terminates only on a complete line that repeats the OPENING
		// code followed by a space. A line whose 4th byte is a space but whose
		// code differs is a continuation, not the end; line[3] == '-' is always a
		// continuation; isPrefix means the line was truncated by the read buffer
		// and so cannot be a terminator.
		if !isPrefix && haveLineCode && line[3] == ' ' && lineCode == openingCode {
			break
		}

		response.WriteString("\n")
	}

	if !haveOpening || openingCode != int(code) {
		return response.String(), &ReturnError{
			rc:      openingCode,
			message: response.String(),
			wantRc:  int(code),
		}
	}

	return response.String(), nil
}
