package zftp

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
	"zftp/internal/utils"
)

// Put a file from the local file system to the remote file system.
//   - if the remote file already exists, it is overwritten.
//   - mode is the transfer mode, either ASCII or binary.
//
// Supports dataset specification as variadic arguments (the same as SetDataSpecs(a ...DataSpec))
func (s *FTPSession) Put(srcLocal string, destRemote string, mode TransferType, a ...DataSpec) error {

	if len(a) > 0 {
		log.Debug("[***] dataset attributes passed to Put()")
		err := s.SetDataSpecs(a...)
		if err != nil {
			return err
		}
	}

	log.Debug("[***] attempting to open source file:", srcLocal)

	file, err := os.Open(srcLocal)
	if err != nil {
		return fmt.Errorf("failed to open source file: %s", err)
	}

	defer func() {
		if cerr := file.Close(); cerr != nil {
			if err != nil {
				err = fmt.Errorf("%s; also failed to close file: %s", err, cerr)
			} else {
				err = fmt.Errorf("failed to close file: %s", cerr)
			}
		}
	}()

	fileInfo, err := file.Stat()
	if err != nil {
		log.Error("Failed to get file stats for:", srcLocal)
		return err
	}
	log.Debug("[***] file stats for", srcLocal, ":")
	log.Debug("[***]   - size in bytes     : ", fileInfo.Size())
	log.Debug("[***]   - file mode         : ", fileInfo.Mode())
	log.Debug("[***]   - modification time : ", fileInfo.ModTime())

	log.Debug("[***] starting transfer to:", destRemote)

	bytesTransferred, err := s.StoreIO(destRemote, file, mode)
	if err != nil {
		return fmt.Errorf("failed to store file: %s", err)
	}

	log.Debugf("[***] successfully transferred %d bytes to %s", bytesTransferred, destRemote)

	return nil
}

// DataSpec is an interface for specifying dataset attributes.
type DataSpec interface {
	getCommand() (string, error)
}

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

func (rec Recfm) getCommand() (string, error) {
	_, ok := isValidRECFM(rec)
	if !ok {
		return "", fmt.Errorf("invalid RECFM value: %s", rec)
	}
	return fmt.Sprintf("RECFM=%s", rec), nil
}

type blksz uint16

func (b blksz) getCommand() (string, error) {
	if b < 1 || b > 32760 {
		return "", fmt.Errorf("blocksize must be between 1 and 32760")
	}
	return fmt.Sprintf("BLKSIZE=%d", b), nil
}

type lrecl uint16

func (r lrecl) getCommand() (string, error) {
	if r < 1 || r > 32760 {
		return "", fmt.Errorf("record length must be between 1 and 32760")
	}
	return fmt.Sprintf("LRECL=%d", r), nil
}

// Lrecl specs for dataset logical record length.
func Lrecl(length uint16) DataSpec {
	return lrecl(length)
}

// Blksize specs for dataset block size.
func Blksize(size uint16) DataSpec {
	return blksz(size)
}

// SetDataSpecs sets the attributes of the dataset being transferred to.
// The attributes are specified as a variadic list of DataSpec.
// Valid attributes are:
//   - Lrecl(length uint16) - record length
//   - Blksize(size uint16) - block size
//   - Recfm(recfm Recfm)   - record format
//
// this function sends a SITE command to the server to set the attributes.
func (s *FTPSession) SetDataSpecs(attributes ...DataSpec) error {

	if len(attributes) == 0 {
		return fmt.Errorf("no attributes specified")
	}

	var cmd strings.Builder
	for _, attr := range attributes {
		a, err := attr.getCommand()
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
