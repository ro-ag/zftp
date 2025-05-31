package zftp

import (
	"compress/gzip"
	"fmt"
	"gopkg.in/ro-ag/zftp.v1/internal/log"
	"gopkg.in/ro-ag/zftp.v1/internal/utils"
	"io"
	"os"
	"path/filepath"
)

// Get retrieves a file from the FTP server and saves it to the local file system.
// If the local file already exists, it is overwritten.
// mode is the transfer mode, either ASCII or binary.
func (s *FTPSession) Get(remote string, localFile string, mode TransferType) error {
	log.Debug("creating local file: ", localFile)
	file, err := os.Create(localFile)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}

	defer func() {
		cerr := file.Close()
		if cerr != nil {
			if err != nil {
				err = fmt.Errorf("%w; also failed to close file: %w", err, cerr)
			} else {
				err = fmt.Errorf("failed to close file: %w", cerr)
			}
		}
	}()

	log.Debug("starting transfer from: ", remote)
	bytesTransferred, _, err := s.RetrieveIO(remote, file, mode)
	if err != nil {
		return fmt.Errorf("failed to retrieve file: %w", err)
	}

	log.Debugf("Successfully transferred %d bytes from %s", bytesTransferred, remote)
	return nil
}

// GetAt retrieves a file from the FTP server starting at a given offset.
// The data is written to the local file beginning at the same offset.
func (s *FTPSession) GetAt(remote string, localFile string, mode TransferType, offset int64) error {
	log.Debug("opening local file: ", localFile)

	file, err := os.OpenFile(localFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}

	if offset == 0 {
		if err = file.Truncate(0); err != nil {
			_ = file.Close()
			return fmt.Errorf("failed to truncate file: %w", err)
		}
	}

	if _, err = file.Seek(offset, io.SeekStart); err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to seek: %w", err)
	}

	defer func() {
		cerr := file.Close()
		if cerr != nil {
			if err != nil {
				err = fmt.Errorf("%w; also failed to close file: %w", err, cerr)
			} else {
				err = fmt.Errorf("failed to close file: %w", cerr)
			}
		}
	}()

	log.Debugf("starting transfer from %s at offset %d", remote, offset)
	bytesTransferred, _, err := s.RetrieveIOAt(remote, file, mode, offset)
	if err != nil {
		return fmt.Errorf("failed to retrieve file: %w", err)
	}

	log.Debugf("successfully transferred %d bytes from %s", bytesTransferred, remote)
	return nil
}

// GetAndGzip retrieves a file from the FTP server and compresses it using gzip.
// The compressed file is saved to the local file system.
// The local file name is the same as the remote file name, with the extension ".gz" appended.
// If the local file already exists, it is overwritten.
// The file is compressed in chunks of 2^32 bytes, so the maximum size of the uncompressed file is 2^32 bytes.
func (s *FTPSession) GetAndGzip(remote string, localFile string, mode TransferType) error {
	if filepath.Ext(localFile) != ".gz" {
		localFile += ".gz"
	}

	log.Debug("creating local file: ", localFile)
	file, err := os.Create(localFile)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}

	gzWriter := gzip.NewWriter(file)

	defer func() {
		err = closeGzHandler(err, gzWriter, file)
	}()

	log.Debug("starting transfer from: ", remote)
	bytesTransferred, _, err := s.RetrieveIO(remote, gzWriter, mode)
	if err != nil {
		return fmt.Errorf("failed to retrieve and compress file: %w", err)
	}

	err = gzWriter.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush gzip writer: %w", err)
	}

	err = gzWriter.Close()
	if err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	log.Debugf("successfully transferred and compressed %d bytes from %s", bytesTransferred, remote)

	err = utils.VerifyGzSize(file, bytesTransferred)
	if err != nil {
		return err
	}

	return nil
}

func closeGzHandler(err error, gzWriter *gzip.Writer, file *os.File) error {
	cerr := gzWriter.Close()
	if cerr != nil {
		if err != nil {
			err = fmt.Errorf("%w; also failed to close gzip writer: %w", err, cerr)
		} else {
			err = fmt.Errorf("failed to close gzip writer: %w", cerr)
		}
	}

	cerr = file.Close()
	if cerr != nil {
		if err != nil {
			err = fmt.Errorf("%w; also failed to close file: %w", err, cerr)
		} else {
			err = fmt.Errorf("failed to close file: %w", cerr)
		}
	}
	return err
}
