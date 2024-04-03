package zftp

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v1/internal/log"
	"gopkg.in/ro-ag/zftp.v1/internal/utils"
	"os"
	"strings"
)

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
		log.Debug("dataset attributes passed to Put()")
		err := s.SetDataSpecs(a...)
		if err != nil {
			return err
		}
	}

	log.Debugf("attempting to open source file: %s", srcLocal)

	file, err := os.Open(srcLocal)
	if err != nil {
		return fmt.Errorf("failed to open source file: %s", err)
	}

	defer func() {
		if cerr := file.Close(); cerr != nil {
			log.Errorf("failed to close file: %s", cerr)
		}
	}()

	fileInfo, err := file.Stat()
	if err != nil {
		log.Error("Failed to get file stats for:", srcLocal)
		return err
	}
	log.Debugf("file stats for %s :", srcLocal)
	log.Debugf("   - size in bytes     : %d", fileInfo.Size())
	log.Debugf("   - file mode         : %s", fileInfo.Mode())
	log.Debugf("   - modification time : %s", fileInfo.ModTime())

	log.Debugf("starting transfer to: %s", destRemote)

	bytesTransferred, _, err := s.StoreIO(destRemote, file, mode)
	if err != nil {
		return fmt.Errorf("failed to store file: %s", err)
	}

	log.Debugf("successfully transferred %d bytes to %s", bytesTransferred, destRemote)

	return nil
}

// DataSpec is an interface for specifying dataset attributes.
type DataSpec interface {
	Apply() (string, error)
}

type Recfm string

const (
	WithRecfmF   Recfm = "F"   // Fixed record length
	WithRecfmFB  Recfm = "FB"  // Fixed length records, Blocked
	WithRecfmFBA Recfm = "FBA" // Fixed length records, Blocked, ASA control characters
	WithRecfmFBM Recfm = "FBM" // Fixed length records, Blocked, Machine control characters
	WithRecfmV   Recfm = "V"   // Variable record length
	WithRecfmVB  Recfm = "VB"  // Variable length records, Blocked
	WithRecfmVBA Recfm = "VBA" // Variable length records, Blocked, ASA control characters
	WithRecfmVBM Recfm = "VBM" // Variable length records, Blocked, Machine control characters
	WithRecfmU   Recfm = "U"   // Undefined record format
	WithRecfmVS  Recfm = "VS"  // Variable record length, Spanned
	WithRecfmVBS Recfm = "VBS" // Variable length records, Blocked, Spanned
)

func isValidRECFM(recfm Recfm) (string, bool) {
	switch recfm {
	case WithRecfmF:
		return "Fixed record length", true
	case WithRecfmFB:
		return "Fixed length records, Blocked", true
	case WithRecfmFBA:
		return "Fixed length records, Blocked, ASA control characters", true
	case WithRecfmFBM:
		return "Fixed length records, Blocked, Machine control characters", true
	case WithRecfmV:
		return "Variable record length", true
	case WithRecfmVB:
		return "Variable length records, Blocked", true
	case WithRecfmVBA:
		return "Variable length records, Blocked, ASA control characters", true
	case WithRecfmVBM:
		return "Variable length records, Blocked, Machine control characters", true
	case WithRecfmU:
		return "Undefined record format", true
	case WithRecfmVS:
		return "Variable record length, Spanned", true
	case WithRecfmVBS:
		return "Variable length records, Blocked, Spanned", true
	default:
		return "", false
	}
}

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
//   - Recfm(recfm Recfm)   - record format
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
		log.Warning(utils.WrapText(msg))
	}
	return nil
}
