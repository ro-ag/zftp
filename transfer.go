// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"context"
	"errors"
	"fmt"
	"gopkg.in/ro-ag/zftp.v2/eol"
	"gopkg.in/ro-ag/zftp.v2/internal/transfer"
	"io"
)

// ErrAsciiResumeUnsupported is returned by the *At transfer methods when a
// byte-offset resume (REST) is requested for an ASCII transfer. A REST argument
// is a byte position, which only has a stable correspondence to a remote position
// in image/binary mode. In TYPE A the server performs end-of-line and codepage
// translation, so resuming by byte offset slices mid-record and silently corrupts
// the data. Resume requires image/binary mode (TypeImage/TypeBinary).
var ErrAsciiResumeUnsupported = errors.New("zftp: byte-offset resume (REST) requires image/binary mode; ASCII transfers cannot be resumed by byte offset")

// guardResume rejects a byte-offset resume that cannot be honored. It returns
// ErrAsciiResumeUnsupported when a positive offset is paired with an ASCII
// transfer type and nil otherwise. The check lives here so every *At entry point
// (RetrieveIOAt, StoreIOAt, GetAt, PutAt) enforces the same rule before any
// local-file or network I/O. Image/binary resume (offset > 0) is unaffected.
func guardResume(t TransferType, offset int64) error {
	if offset > 0 && t.IsAscii() {
		return ErrAsciiResumeUnsupported
	}
	return nil
}

// TransferType is the FTP representation type for a transfer. It is a concrete
// enum (not an interface): callers pass one of the exported values rather than
// implementing it.
type TransferType uint8

const (
	// TypeAscii selects ASCII mode (TYPE A), with end-of-line conversion.
	TypeAscii TransferType = 'A'
	// TypeImage selects binary/image mode (TYPE I), byte-for-byte.
	TypeImage TransferType = 'I'
	// TypeBinary is an alias for TypeImage.
	TypeBinary = TypeImage
)

// strCommand returns the FTP command string for the transfer type.
func (t TransferType) strCommand() string {
	return fmt.Sprintf("TYPE %c", t)
}

// Name returns the human-readable name of the transfer type.
func (t TransferType) Name() string {
	if t.IsAscii() {
		return "ASCII"
	}
	return "BINARY"
}

// IsAscii reports whether the transfer type is ASCII.
func (t TransferType) IsAscii() bool {
	return t == TypeAscii
}

// IsBinary reports whether the transfer type is binary/image.
func (t TransferType) IsBinary() bool {
	return t == TypeBinary
}

// SetType sets the transfer type and stores it in the FTPSession.
func (s *FTPSession) SetType(t TransferType) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.setTypeLocked(t)
}

// setTypeLocked issues the TYPE command and records the new transfer type on
// success. The caller must hold s.mu.
func (s *FTPSession) setTypeLocked(t TransferType) error {
	_, err := s.sendLocked(context.Background(), CodeCmdOK, t.strCommand())
	if err == nil {
		s.currType.Store(uint32(t))
	}
	return err
}

// currentType returns the session's current transfer type. It is safe to call
// without holding s.mu.
func (s *FTPSession) currentType() TransferType {
	return TransferType(s.currType.Load())
}

// transfer is a helper function that performs a data transfer.
// If offset is greater than zero, a REST command is issued before
// starting the transfer to resume at the given byte position.
func (s *FTPSession) transfer(t transfer.DataTransfer, remote string, offset int64) (int64, string, error) {

	port, err := s.SetPassiveMode()
	if err != nil {
		return 0, "", err
	}

	child, err := s.newChildConnection(port)
	if err != nil {
		return 0, "", err
	}
	defer func(child *childConnection) {
		if !child.IsClosed() {
			if err = child.Close(); err != nil {
				s.log.Error(err)
			}
		}
	}(child)

	if offset > 0 {
		if _, err := s.SendCommand(CodeNeedInfo, "REST", fmt.Sprintf("%d", offset)); err != nil {
			return 0, "", err
		}
	}

	msg1, err := s.SendCommand(CodeListOK, t.Command(), remote)
	if err != nil {
		return 0, msg1, err
	}

	sz, err := t.Transfer(child)
	if err != nil {
		// A data-stream failure leaves the transfer's terminal control reply
		// unconsumed, desynchronizing the control stream; close the session so it
		// is not reused one reply out of phase.
		_ = s.Close()
		return sz, msg1, err
	}

	msg2, err := s.confirmData(child)
	if err != nil {
		return sz, msg1, fmt.Errorf("error while checking last response: %w", err)
	}

	return sz, fmt.Sprintf("%s\n%s", msg1, msg2), nil
}

// confirmData finalizes a data transfer. The caller must have already drained the
// data connection to EOF — closing it with unread bytes would make the local TCP
// stack emit an RST. It closes the data connection and reads the terminal reply on
// the control connection; that read is timeout-bounded by checkLast because z/OS
// sends the reply asynchronously to the data close and it can be lost.
func (s *FTPSession) confirmData(child *childConnection) (string, error) {
	if err := child.Close(); err != nil {
		return "", err
	}
	return s.checkLast(CodeFileActionOK)
}

// StoreIO stores the contents of the reader to the remote file in the specified
// mode and returns the number of bytes transferred.
//
// The original transfer type is restored to the previous value after the transfer,
// supports ASCII and binary/Image transfers
func (s *FTPSession) StoreIO(remote string, src io.Reader, t TransferType) (int64, string, error) {

	current := s.currentType()
	if err := s.SetType(t); err != nil {
		return 0, "", err
	}

	var format transfer.DataTransfer

	if t.IsAscii() {
		format = transfer.NewStoreAscii(src)
	} else {
		format = transfer.NewStore(src)
	}

	sz, msg, err := s.transfer(format, remote, 0)
	if err != nil {
		return sz, msg, err
	}

	if err = s.SetType(current); err != nil {
		return sz, msg, fmt.Errorf("error while setting back the transfer type: %w", err)
	}

	return sz, msg, err
}

// RetrieveIO retrieves the contents of the remote file and writes it to the writer
// The transfer type is restored to the previous value after the transfer
// supports ASCII and binary/Image transfers
func (s *FTPSession) RetrieveIO(remote string, dest io.Writer, t TransferType) (int64, string, error) {
	current := s.currentType()
	if t.IsAscii() {
		if err := s.SetStatusOf().SBSendEol(eol.System); err != nil {
			return 0, "", err
		}
	}
	if err := s.SetType(t); err != nil {
		return 0, "", err
	}

	sz, msg, err := s.transfer(transfer.NewRetrieve(dest), remote, 0)
	if err != nil {
		return sz, msg, err
	}

	if err = s.SetType(current); err != nil {
		return sz, msg, fmt.Errorf("error while setting back the transfer type: %w", err)
	}

	return sz, msg, err
}

// StoreIOAt performs a store operation starting at the given offset on the remote file.
// The transfer type is restored to the previous value after the transfer.
//
// Resume requires image/binary mode: a positive offset combined with TypeAscii
// returns ErrAsciiResumeUnsupported before any I/O, because in ASCII mode the
// server's EOL/codepage translation makes a byte offset corrupt the data.
func (s *FTPSession) StoreIOAt(remote string, src io.Reader, t TransferType, offset int64) (int64, string, error) {

	if err := guardResume(t, offset); err != nil {
		return 0, "", err
	}

	current := s.currentType()
	if err := s.SetType(t); err != nil {
		return 0, "", err
	}

	var format transfer.DataTransfer

	if t.IsAscii() {
		format = transfer.NewStoreAscii(src)
	} else {
		format = transfer.NewStore(src)
	}

	sz, msg, err := s.transfer(format, remote, offset)
	if err != nil {
		return sz, msg, err
	}

	if err = s.SetType(current); err != nil {
		return sz, msg, fmt.Errorf("error while setting back the transfer type: %w", err)
	}

	return sz, msg, err
}

// RetrieveIOAt performs a retrieve operation starting from the given offset of the remote file.
// The transfer type is restored to the previous value after the transfer.
//
// Resume requires image/binary mode: a positive offset combined with TypeAscii
// returns ErrAsciiResumeUnsupported before any I/O, because in ASCII mode the
// server's EOL/codepage translation makes a byte offset corrupt the data.
func (s *FTPSession) RetrieveIOAt(remote string, dest io.Writer, t TransferType, offset int64) (int64, string, error) {
	if err := guardResume(t, offset); err != nil {
		return 0, "", err
	}
	current := s.currentType()
	if t.IsAscii() {
		if err := s.SetStatusOf().SBSendEol(eol.System); err != nil {
			return 0, "", err
		}
	}
	if err := s.SetType(t); err != nil {
		return 0, "", err
	}

	sz, msg, err := s.transfer(transfer.NewRetrieve(dest), remote, offset)
	if err != nil {
		return sz, msg, err
	}

	if err = s.SetType(current); err != nil {
		return sz, msg, fmt.Errorf("error while setting back the transfer type: %w", err)
	}

	return sz, msg, err
}
