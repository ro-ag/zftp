package zftp

import (
	"bufio"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"strconv"
	"strings"
)

//go:generate stringer -type=ReturnCode
type ReturnCode int

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

type ReturnError struct {
	rc      int
	message string
	wantRc  int
}

func (e *ReturnError) Got() ReturnCode {
	return ReturnCode(e.rc)
}

func (e *ReturnError) Error() string {
	return fmt.Sprintf("FTP response code: got %d, want %d, message: %s", e.rc, e.wantRc, e.message)
}

func (code ReturnCode) CheckCode(reader *bufio.Reader) (msg string, err error) {

	var (
		response     strings.Builder
		receivedCode = 0
	)

	for {
		line, isPrefix, err := reader.ReadLine()
		if err != nil {
			// If ReadLine returns an io.EOF error, we return what we have
			if err == io.EOF {
				break
			}
			return "", err
		}

		log.Debugf("[res] %s", line)

		// We only check the response code if we've read a complete line
		if len(line) >= 4 {

			tempReceivedCode, atoiErr := strconv.Atoi(string(line[:3]))
			if atoiErr != nil {
				log.WithFields(log.Fields{"error": atoiErr}).Error("Error converting response code to integer")
			} else {
				receivedCode = tempReceivedCode
			}

			if receivedCode == int(code) {
				response.Write(line[4:])
			} else {
				response.Write(line) // Keep thew whole response to isolate the error
			}

			if isPrefix {
				// If the line is too long to fit in the buffer, we read the rest of the line
				response.WriteString("\n")
				continue
			}

			if line[3] == '-' {
				response.WriteString("\n")
				continue
			}

			if line[3] == ' ' {
				// If we've found the expected response code and the line ends with a space,
				// we can return that the code is OK, along with the response
				break
			}
		}
	}

	if receivedCode != int(code) {
		return response.String(), &ReturnError{
			rc:      receivedCode,
			message: response.String(),
			wantRc:  int(code),
		}
	}

	return response.String(), nil
}
