package zftp

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v1/eol"
	"gopkg.in/ro-ag/zftp.v1/internal/log"
	"gopkg.in/ro-ag/zftp.v1/internal/transfer"
	"io"
)

type TransferType interface {
	strCommand() string
	IsAscii() bool
	IsBinary() bool
	Name() string
}

type transferType uint8

const (
	TypeAscii  transferType = 'A'
	TypeImage  transferType = 'I'
	TypeBinary              = TypeImage
)

// StrCommand returns the command string for the transfer type
func (t transferType) strCommand() string {
	return fmt.Sprintf("TYPE %c", t)
}

// Name returns the name of the transfer type
func (t transferType) Name() string {
	if t.IsAscii() {
		return "ASCII"
	}
	return "BINARY"
}

// SetType sets the transfer type and stores it in the FTPSession
func (s *FTPSession) SetType(t TransferType) error {
	_, err := s.SendCommand(CodeCmdOK, t.strCommand())
	if err == nil {
		s.currType = t
	}
	return err
}

// IsAscii returns true if the transfer type is ASCII
func (t transferType) IsAscii() bool {
	return t == TypeAscii
}

// IsBinary returns true if the transfer type is binary
func (t transferType) IsBinary() bool {
	return t == TypeBinary
}

// transfer is a helper function that performs a data transfer
func (s *FTPSession) transfer(t transfer.DataTransfer, remote string) (int64, string, error) {

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
				log.Error(err)
			}
		}
	}(child)

	msg1, err := s.SendCommand(CodeListOK, t.Command(), remote)
	if err != nil {
		return 0, "", err
	}

	sz, err := t.Transfer(child)
	if err != nil {
		return sz, "", err
	}

	err = child.Close()
	if err != nil {
		return sz, "", err
	}

	msg2, err := s.checkLast(CodeFileActionOK)
	if err != nil {
		return sz, "", fmt.Errorf("error while checking last response: %w", err)
	}

	return sz, fmt.Sprintf("%s\n%s", msg1, msg2), nil
}

// StoreIO stores the contents of the reader to the remote file in the specified
// mode and returns the number of bytes transferred.
//
// The original transfer type is restored to the previous value after the transfer,
// supports ASCII and binary/Image transfers
func (s *FTPSession) StoreIO(remote string, src io.Reader, t TransferType) (int64, string, error) {

	current := s.currType
	if err := s.SetType(t); err != nil {
		return 0, "", err
	}

	var format transfer.DataTransfer

	if t.IsAscii() {
		format = transfer.NewStoreAscii(src)
	} else {
		format = transfer.NewStore(src)
	}

	sz, msg, err := s.transfer(format, remote)
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
	current := s.currType
	if t.IsAscii() {
		if err := s.SetStatusOf().SBSendEol(eol.System); err != nil {
			return 0, "", err
		}
	}
	if err := s.SetType(t); err != nil {
		return 0, "", err
	}

	sz, msg, err := s.transfer(transfer.NewRetrieve(dest), remote)
	if err != nil {
		return sz, msg, err
	}

	if err = s.SetType(current); err != nil {
		return sz, msg, fmt.Errorf("error while setting back the transfer type: %w", err)
	}

	return sz, msg, err
}
