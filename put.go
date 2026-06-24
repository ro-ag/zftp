// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v2/internal/utils"
	"io"
	"os"
	"strings"
)

// Block-size bounds accepted by WithBlkSize for dataset transfers.
const (
	MinBlockSize = 1
	MaxBlockSize = 32768
)

// Put a file from the local file system to the remote file system.
//   - if the remote file already exists, it is overwritten.
//   - mode is the transfer mode, either ASCII or binary.
//
// Supports dataset specification as variadic arguments (the same as SetDataSpecs(a ...DataSpec))
func (s *FTPSession) Put(srcLocal string, destRemote string, mode TransferType, a ...DataSpec) error {

	if len(a) > 0 {
		s.log.Debug("dataset attributes passed to Put()")
		err := s.SetDataSpecs(a...)
		if err != nil {
			return err
		}
	}

	s.log.Debugf("attempting to open source file: %s", srcLocal)

	file, err := os.Open(srcLocal)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}

	defer func() {
		if cerr := file.Close(); cerr != nil {
			s.log.Errorf("failed to close file: %s", cerr)
		}
	}()

	fileInfo, err := file.Stat()
	if err != nil {
		s.log.Error("Failed to get file stats for:", srcLocal)
		return err
	}
	s.log.Debugf("file stats for %s :", srcLocal)
	s.log.Debugf("   - size in bytes     : %d", fileInfo.Size())
	s.log.Debugf("   - file mode         : %s", fileInfo.Mode())
	s.log.Debugf("   - modification time : %s", fileInfo.ModTime())

	s.log.Debugf("starting transfer to: %s", destRemote)

	bytesTransferred, err := s.StoreIO(destRemote, file, mode)
	if err != nil {
		return fmt.Errorf("failed to store file: %w", err)
	}

	s.log.Debugf("successfully transferred %d bytes to %s", bytesTransferred, destRemote)

	return nil
}

// PutAt resumes uploading a file starting from the given offset.
// If dataset attributes are provided, they will be applied before the transfer.
//
// Resume requires image/binary mode: a positive offset combined with TypeAscii
// returns ErrAsciiResumeUnsupported before the source file is opened or seeked and
// before any SITE/REST is sent, because in ASCII mode the server's EOL/codepage
// translation makes a byte offset corrupt the data.
func (s *FTPSession) PutAt(srcLocal string, destRemote string, mode TransferType, offset int64, a ...DataSpec) error {

	if err := guardResume(mode, offset); err != nil {
		return err
	}

	if len(a) > 0 {
		s.log.Debug("dataset attributes passed to PutAt()")
		if err := s.SetDataSpecs(a...); err != nil {
			return err
		}
	}

	s.log.Debugf("attempting to open source file: %s", srcLocal)

	file, err := os.Open(srcLocal)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}

	if _, err = file.Seek(offset, io.SeekStart); err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to seek: %w", err)
	}

	defer func() {
		if cerr := file.Close(); cerr != nil {
			s.log.Errorf("failed to close file: %s", cerr)
		}
	}()

	fileInfo, err := file.Stat()
	if err != nil {
		s.log.Error("Failed to get file stats for:", srcLocal)
		return err
	}
	s.log.Debugf("file stats for %s :", srcLocal)
	s.log.Debugf("   - size in bytes     : %d", fileInfo.Size())
	s.log.Debugf("   - file mode         : %s", fileInfo.Mode())
	s.log.Debugf("   - modification time : %s", fileInfo.ModTime())

	s.log.Debugf("starting transfer to: %s at offset %d", destRemote, offset)

	bytesTransferred, err := s.StoreIOAt(destRemote, file, mode, offset)
	if err != nil {
		return fmt.Errorf("failed to store file: %w", err)
	}

	s.log.Debugf("successfully transferred %d bytes to %s", bytesTransferred, destRemote)

	return nil
}

// DataSpec is an interface for specifying dataset attributes.
type DataSpec interface {
	Apply() (string, error)
}

// Recfm is a z/OS record format (RECFM) usable as a DataSpec via the Recfm…
// constants (e.g. RecfmFB); its Apply renders the SITE RECFM= subcommand.
type Recfm string

const (
	RecfmF   Recfm = "F"   // Fixed record length
	RecfmFB  Recfm = "FB"  // Fixed length records, Blocked
	RecfmFBA Recfm = "FBA" // Fixed length records, Blocked, ASA control characters
	RecfmFBM Recfm = "FBM" // Fixed length records, Blocked, Machine control characters
	RecfmV   Recfm = "V"   // Variable record length
	RecfmVB  Recfm = "VB"  // Variable length records, Blocked
	RecfmVBA Recfm = "VBA" // Variable length records, Blocked, ASA control characters
	RecfmVBM Recfm = "VBM" // Variable length records, Blocked, Machine control characters
	RecfmU   Recfm = "U"   // Undefined record format
	RecfmVS  Recfm = "VS"  // Variable record length, Spanned
	RecfmVBS Recfm = "VBS" // Variable length records, Blocked, Spanned
)

func isValidRECFM(recfm Recfm) (string, bool) {
	switch recfm {
	case RecfmF:
		return "Fixed record length", true
	case RecfmFB:
		return "Fixed length records, Blocked", true
	case RecfmFBA:
		return "Fixed length records, Blocked, ASA control characters", true
	case RecfmFBM:
		return "Fixed length records, Blocked, Machine control characters", true
	case RecfmV:
		return "Variable record length", true
	case RecfmVB:
		return "Variable length records, Blocked", true
	case RecfmVBA:
		return "Variable length records, Blocked, ASA control characters", true
	case RecfmVBM:
		return "Variable length records, Blocked, Machine control characters", true
	case RecfmU:
		return "Undefined record format", true
	case RecfmVS:
		return "Variable record length, Spanned", true
	case RecfmVBS:
		return "Variable length records, Blocked, Spanned", true
	default:
		return "", false
	}
}

// Apply renders the SITE subcommand for this record format, or an error if the
// value is not a recognized RECFM.
func (rec Recfm) Apply() (string, error) {
	_, ok := isValidRECFM(rec)
	if !ok {
		return "", fmt.Errorf("invalid RECFM value: %s", rec)
	}
	return fmt.Sprintf("RECFM=%s", rec), nil
}

type blksz uint16

func (b blksz) Apply() (string, error) {
	if b < MinBlockSize || b > MaxBlockSize {
		return "", fmt.Errorf("blocksize must be between %d and %d", MinBlockSize, MaxBlockSize)
	}
	return fmt.Sprintf("BLKSIZE=%d", b), nil
}

type lrecl uint16

func (r lrecl) Apply() (string, error) {
	if r < 1 || r > 32760 {
		return "", fmt.Errorf("record length must be between 1 and 32760")
	}
	return fmt.Sprintf("LRECL=%d", r), nil
}

// Compile-time checks that the DataSpec implementations satisfy the interface.
var (
	_ DataSpec = Recfm("")
	_ DataSpec = blksz(0)
	_ DataSpec = lrecl(0)
)

// WithLrecl specs for dataset logical record length.
func WithLrecl(length uint16) DataSpec {
	return lrecl(length)
}

// WithBlkSize specs for dataset block size.
func WithBlkSize(size uint16) DataSpec {
	return blksz(size)
}

// SetDataSpecs sets the attributes of the dataset being transferred to.
// The attributes are specified as a variadic list of DataSpec.
// Valid attributes are:
//   - WithLrecl(length uint16) - record length
//   - WithBlkSize(size uint16) - block size
//   - a Recfm constant (e.g. RecfmFB) - record format
//
// this function sends a SITE command to the server to set the attributes.
func (s *FTPSession) SetDataSpecs(attributes ...DataSpec) error {

	if len(attributes) == 0 {
		return fmt.Errorf("no attributes specified")
	}

	var cmd strings.Builder
	for _, attr := range attributes {
		a, err := attr.Apply()
		if err != nil {
			return err
		}

		cmd.WriteString(a)
		cmd.WriteString(" ")
	}

	msg, err := s.Site(cmd.String())
	if err != nil {
		return err
	}
	if msg != "SITE command was accepted" {
		s.log.Warning(utils.WrapText(msg))
	}
	return nil
}
