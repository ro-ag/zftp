// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v2/internal/utils"
	"regexp"
	"strconv"
)

var (
	recFmt = regexp.MustCompile(`^Record\s+format\s+(\w+)\s*,\s*Lrecl:\s*(\d+)\s*,\s*Blocksize:\s*(\d+)`)
)

// ServerStatus queries individual z/OS server status values via the XSTA
// command. It is returned by FTPSession.StatusOf; each method issues one query
// and parses the reply. Obtain it from the session rather than constructing it.
type ServerStatus struct {
	xstat func(string) (string, error)
}

// ASATrans reports whether ASA carriage-control handling is enabled (XSTA ASATRANS).
func (s *ServerStatus) ASATrans() (string, error) {
	resp, err := s.xstat("ASATrans")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// AutoMount reports whether volumes are automatically mounted for allocation (XSTA AUTOMOUNT).
func (s *ServerStatus) AutoMount() (string, error) {
	resp, err := s.xstat("AUTOMount")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// AutoRecall reports whether migrated datasets are automatically recalled (XSTA AUTORECALL).
func (s *ServerStatus) AutoRecall() (string, error) {
	resp, err := s.xstat("AUTORecall")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// BLocks reports whether primary/secondary space is allocated in blocks (XSTA BLOCKS).
func (s *ServerStatus) BLocks() (string, error) {
	resp, err := s.xstat("BLocks")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// BlockSize returns the block size (BLKSIZE) from the record-format status line (XSTA BLOCKSIZE).
func (s *ServerStatus) BlockSize() (int, error) {
	resp, err := s.xstat("BLOCKSIze")
	if err != nil {
		return 0, err
	}

	m := recFmt.FindStringSubmatch(resp)
	if len(m) < 4 {
		return 0, fmt.Errorf("unexpected response: %s", resp)
	}

	return strconv.Atoi(m[3])
}

// BufNo returns the number of access-method buffers configured (XSTA BUFNO).
func (s *ServerStatus) BufNo() (int, error) {
	resp, err := s.xstat("BUfno")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

// CheckpointInterval returns the checkpoint interval, in records, for the data connection (XSTA CHKPTINT).
func (s *ServerStatus) CheckpointInterval() (int, error) {
	resp, err := s.xstat("CHKptint")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

// ConditionDisposition returns the disposition applied to a dataset when a transfer fails (XSTA CONDDISP).
func (s *ServerStatus) ConditionDisposition() (string, error) {
	resp, err := s.xstat("CONDdisp")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// Cylinders reports whether primary/secondary space is allocated in cylinders (XSTA CYLINDERS).
func (s *ServerStatus) Cylinders() (string, error) {
	resp, err := s.xstat("CYlinders")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// DataClass returns the SMS data class used for new dataset allocation (XSTA DATACLASS).
func (s *ServerStatus) DataClass() (string, error) {
	resp, err := s.xstat("DATAClass")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// DataKeepAlive returns the data-connection TCP keep-alive interval, in seconds (XSTA DATAKEEPALIVE).
func (s *ServerStatus) DataKeepAlive() (int, error) {
	resp, err := s.xstat("DATAKEEPALIVE")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

// DatasetMode reports whether the server is in dataset mode rather than directory mode (XSTA DATASETMODE).
func (s *ServerStatus) DatasetMode() (string, error) {
	resp, err := s.xstat("DATASetmode")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// DB2 returns the DB2 subsystem name used for SQL queries (XSTA DB2).
func (s *ServerStatus) DB2() (string, error) {
	resp, err := s.xstat("DB2")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// DoubleByteSubstitution reports whether double-byte substitution is enabled (XSTA DBSUB).
func (s *ServerStatus) DoubleByteSubstitution() (bool, error) {
	resp, err := s.xstat("DBSUB")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

// DCBDSN returns the dataset whose DCB attributes model new allocations (XSTA DCBDSN).
func (s *ServerStatus) DCBDSN() (string, error) {
	resp, err := s.xstat("DCBDSN")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// Destination returns the SYSOUT destination for jobs and reports (XSTA DEST).
func (s *ServerStatus) Destination() (string, error) {
	resp, err := s.xstat("DESt")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// Directory returns the number of directory blocks allocated for a new PDS (XSTA DIRECTORY).
func (s *ServerStatus) Directory() (string, error) {
	resp, err := s.xstat("Directory")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// DirectoryMode reports whether the server is in directory mode rather than dataset mode (XSTA DIRECTORYMODE).
func (s *ServerStatus) DirectoryMode() (string, error) {
	resp, err := s.xstat("DIRECTORYMode")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// DSNType returns the data-set name type (e.g. PDS, LIBRARY, BASIC) for new allocations (XSTA DSNTYPE).
func (s *ServerStatus) DSNType() (string, error) {
	resp, err := s.xstat("DSNTYPE")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// DSWaitTime returns how long, in minutes, FTP waits for a held dataset before failing (XSTA DSWAITTIME).
func (s *ServerStatus) DSWaitTime() (int, error) {
	resp, err := s.xstat("DSWAITTIME")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

// EATTR returns the extended-attributes setting for new dataset allocation (XSTA EATTR).
func (s *ServerStatus) EATTR() (string, error) {
	resp, err := s.xstat("EATTR")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// Encoding returns the encoding (SBCS or MBCS) used for the data connection (XSTA ENCODING).
func (s *ServerStatus) Encoding() (string, error) {
	resp, err := s.xstat("ENCODING")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// FifoIoTime returns the time-out, in seconds, for a single read/write on a z/OS FIFO (XSTA FIFOIOTIME).
func (s *ServerStatus) FifoIoTime() (int, error) {
	resp, err := s.xstat("FIFOIOTIME")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

// FifoOpenTime returns the time-out, in seconds, for opening a z/OS FIFO (XSTA FIFOOPENTIME).
func (s *ServerStatus) FifoOpenTime() (int, error) {
	resp, err := s.xstat("FIFOOPENTIME")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

// FileType returns the current file type — SEQ, JES, or SQL (XSTA FILETYPE).
func (s *ServerStatus) FileType() (string, error) {
	resp, err := s.xstat("FileType")
	if err != nil {
		return "", err
	}
	regx := regexp.MustCompile(`^FileType\s+(\w+).*$`)
	ft := regx.FindStringSubmatch(resp)
	if len(ft) < 2 {
		return "", fmt.Errorf("could not parse file type")
	}
	return ft[1], nil
}

// FTPKeepAlive returns the control-connection TCP keep-alive interval, in seconds (XSTA FTPKEEPALIVE).
func (s *ServerStatus) FTPKeepAlive() (int, error) {
	resp, err := s.xstat("FTpkeepalive")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

// InactiveTime returns the inactivity time-out, in seconds, before the session is closed (XSTA INACTIVETIME).
func (s *ServerStatus) InactiveTime() (int, error) {
	resp, err := s.xstat("INactivetime")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

// ISPFStats reports whether ISPF statistics are maintained for PDS members (XSTA ISPFSTATS).
func (s *ServerStatus) ISPFStats() (bool, error) {
	resp, err := s.xstat("ISPFSTATS")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

// JesEntryLimit returns the maximum number of JES entries a query may return (XSTA JESENTRYLIMIT).
func (s *ServerStatus) JesEntryLimit() (int, error) {
	resp, err := s.xstat("JESENTRYLimit")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

// JesGetByDSN reports whether retrieving a job's output by DSN is enabled (XSTA JESGETBYDSN).
func (s *ServerStatus) JesGetByDSN() (bool, error) {
	resp, err := s.xstat("JESGETBYDSN")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

// JesJobName returns the job-name filter applied to JES queries (XSTA JESJOBNAME).
func (s *ServerStatus) JesJobName() (string, error) {
	resp, err := s.xstat("JESJOBName")
	if err != nil {
		return "", err
	}
	return utils.LastText(resp), nil
}

// JesLrecl returns the record length used for jobs submitted to the internal reader (XSTA JESLRECL).
func (s *ServerStatus) JesLrecl() (int, error) {
	resp, err := s.xstat("JESLrecl")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

// JesOwner returns the owner filter applied to JES queries (XSTA JESOWNER).
func (s *ServerStatus) JesOwner() (string, error) {
	resp, err := s.xstat("JESOwner")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// JesRecfm returns the record format used for jobs submitted to the internal reader (XSTA JESRECFM).
func (s *ServerStatus) JesRecfm() (string, error) {
	resp, err := s.xstat("JESRecfm")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// JesStatus returns the job-status filter applied to JES queries (XSTA JESSTATUS).
func (s *ServerStatus) JesStatus() (string, error) {
	resp, err := s.xstat("JESSTatus")
	if err != nil {
		return "", err
	}
	return utils.LastText(resp), nil
}

// ListLevel returns the LISTLEVEL controlling the column layout of dataset listings (XSTA LISTLEVEL).
func (s *ServerStatus) ListLevel() (int, error) {
	resp, err := s.xstat("LISTLEVEL")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

// ListSubDir reports whether LIST recurses into HFS subdirectories (XSTA LISTSUBDIR).
func (s *ServerStatus) ListSubDir() (bool, error) {
	resp, err := s.xstat("LISTSUBdir")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

// Lrecl returns the logical record length (LRECL) from the record-format status line (XSTA LRECL).
func (s *ServerStatus) Lrecl() (int, error) {
	resp, err := s.xstat("Lrecl")
	if err != nil {
		return 0, err
	}
	m := recFmt.FindStringSubmatch(resp)
	if len(m) < 4 {
		return 0, fmt.Errorf("could not parse Lrecl: %s", resp)
	}
	// recFmt groups: m[1]=Recfm, m[2]=Lrecl, m[3]=Blocksize.
	return strconv.Atoi(m[2])
}

// MBDataConn returns the multibyte data-connection codepage pair (network, host) (XSTA MBDATACONN).
func (s *ServerStatus) MBDataConn() (string, error) {
	resp, err := s.xstat("MBDATACONN")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// MBRequireLastEol reports whether a final end-of-line is required on inbound multibyte data (XSTA MBREQUIRELASTEOL).
func (s *ServerStatus) MBRequireLastEol() (bool, error) {
	resp, err := s.xstat("MBREQUIRELASTEOL")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

var eolFmt = regexp.MustCompile(`uses\s+(\w+)\s+line\s+terminator$`)

// MBSendEol returns the end-of-line sequence appended to outbound multibyte data (XSTA MBSENDEOL).
func (s *ServerStatus) MBSendEol() (string, error) {
	resp, err := s.xstat("MBSENDEOL")
	if err != nil {
		return "", err
	}
	m := eolFmt.FindStringSubmatch(resp)
	if len(m) < 2 {
		return "", fmt.Errorf("could not parse MBSENDEOL")
	}
	return m[1], nil
}

// MgmtClass returns the SMS management class used for new dataset allocation (XSTA MGMTCLASS).
func (s *ServerStatus) MgmtClass() (string, error) {
	resp, err := s.xstat("MGmtclass")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// MigrateVol returns the volume serial reported for a migrated dataset (XSTA MIGRATEVOL).
func (s *ServerStatus) MigrateVol() (string, error) {
	resp, err := s.xstat("MIGratevol")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// PDSType reports whether new partitioned datasets are created as PDS or PDSE (XSTA PDSTYPE).
func (s *ServerStatus) PDSType() (string, error) {
	resp, err := s.xstat("PDSTYPE")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// Primary returns the primary space allocation for new datasets (XSTA PRIMARY).
func (s *ServerStatus) Primary() (string, error) {
	resp, err := s.xstat("Primary")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// QuotesOverride reports whether a leading quote overrides the working-directory prefix (XSTA QUOTESOVERRIDE).
func (s *ServerStatus) QuotesOverride() (string, error) {
	resp, err := s.xstat("QUOtesoverride")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// RDW reports whether variable-record descriptor words are retained as data (XSTA RDW).
func (s *ServerStatus) RDW() (string, error) {
	resp, err := s.xstat("RDW")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// ReadTapeFormat returns the record format used when reading a tape dataset (XSTA READTAPEFORMAT).
func (s *ServerStatus) ReadTapeFormat() (string, error) {
	resp, err := s.xstat("READTAPEFormat")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// Recfm returns the record format (RECFM) from the record-format status line (XSTA RECFM).
func (s *ServerStatus) Recfm() (string, error) {
	resp, err := s.xstat("Recfm")
	if err != nil {
		return "", err
	}
	m := recFmt.FindStringSubmatch(resp)
	if len(m) < 4 {
		return "", fmt.Errorf("could not parse Recfm")
	}
	return m[1], nil
}

// RetPD returns the retention period, in days, for new datasets (XSTA RETPD).
func (s *ServerStatus) RetPD() (int, error) {
	resp, err := s.xstat("RetPD")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

// SBDataConn returns the single-byte data-connection codepage reported by the
// server (e.g. IBM-1047). SBDATACONN is a codepage string, not an integer, so it
// is returned as a string (mirroring MBDataConn).
func (s *ServerStatus) SBDataConn() (string, error) {
	resp, err := s.xstat("SBDataConn")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// SBSendEol returns the end-of-line sequence appended to outbound single-byte data (XSTA SBSENDEOL).
func (s *ServerStatus) SBSendEol() (string, error) {
	resp, err := s.xstat("SBSENDEOL")
	if err != nil {
		return "", err
	}
	m := eolFmt.FindStringSubmatch(resp)
	if len(m) < 2 {
		return "", fmt.Errorf("could not parse SBSENDEOL")
	}
	return m[1], nil
}

// SBSub reports whether single-byte substitution is enabled (XSTA SBSUB).
func (s *ServerStatus) SBSub() (bool, error) {
	resp, err := s.xstat("SBSUB")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

// SBSubChar returns the substitution character used for untranslatable single-byte data (XSTA SBSUBCHAR).
func (s *ServerStatus) SBSubChar() (string, error) {
	resp, err := s.xstat("SBSUBCHAR")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// Secondary returns the secondary space allocation for new datasets (XSTA SECONDARY).
func (s *ServerStatus) Secondary() (string, error) {
	resp, err := s.xstat("SECondary")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// SPRead returns the SPREAD setting controlling multi-volume primary allocation (XSTA SPREAD).
func (s *ServerStatus) SPRead() (string, error) {
	resp, err := s.xstat("SPRead")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// SQLCol returns the SQL column-naming setting for query output (XSTA SQLCOL).
func (s *ServerStatus) SQLCol() (string, error) {
	resp, err := s.xstat("SQLCol")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// StorageClass returns the SMS storage class used for new dataset allocation (XSTA STORCLASS).
func (s *ServerStatus) StorageClass() (string, error) {
	resp, err := s.xstat("STOrclass")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// TlsRfcLevel returns the level of RFC 4217 (FTP-over-TLS) support reported (XSTA TLSRFCLEVEL).
func (s *ServerStatus) TlsRfcLevel() (string, error) {
	resp, err := s.xstat("TLSRFCLEVEL")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// Tracks reports whether primary/secondary space is allocated in tracks (XSTA TRACKS).
func (s *ServerStatus) Tracks() (string, error) {
	resp, err := s.xstat("TRacks")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// TrailingBlanks reports whether trailing blanks in fixed-length records are preserved (XSTA TRAILINGBLANKS).
func (s *ServerStatus) TrailingBlanks() (string, error) {
	resp, err := s.xstat("TRAILingblanks")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// Truncate reports whether records longer than LRECL are truncated rather than failing (XSTA TRUNCATE).
func (s *ServerStatus) Truncate() (string, error) {
	resp, err := s.xstat("TRUNcate")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// UCount returns the number of devices (unit count) for a new allocation (XSTA UCOUNT).
func (s *ServerStatus) UCount() (int, error) {
	resp, err := s.xstat("UCOUNT")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

// UCSHostCS returns the host character set used for Unicode conversion (XSTA UCSHOSTCS).
func (s *ServerStatus) UCSHostCS() (string, error) {
	resp, err := s.xstat("UCSHOSTCS")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// UCSSub reports whether substitution is used for untranslatable Unicode characters (XSTA UCSSUB).
func (s *ServerStatus) UCSSub() (string, error) {
	resp, err := s.xstat("UCSSUB")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// UCSTrunc reports whether Unicode data is truncated on a conversion error (XSTA UCSTRUNC).
func (s *ServerStatus) UCSTrunc() (string, error) {
	resp, err := s.xstat("UCSTRUNC")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// UMask returns the file-mode creation mask reported by the server. z/OS reports
// UMASK in octal, so the value is parsed base 8 and returned as its integer value
// (e.g. "022" -> 18).
func (s *ServerStatus) UMask() (int, error) {
	resp, err := s.xstat("UMask")
	if err != nil {
		return 0, err
	}
	w := utils.LastWord(resp)
	n, err := strconv.ParseInt(w, 8, 0)
	if err != nil {
		return 0, fmt.Errorf("could not parse octal UMask %q: %w", w, err)
	}
	return int(n), nil
}

// UnicodeFileSystemBOM reports whether a byte-order mark is written to Unicode HFS files (XSTA UNICODEFILESYSTEMBOM).
func (s *ServerStatus) UnicodeFileSystemBOM() (string, error) {
	resp, err := s.xstat("UNICODEFILESYSTEMBOM")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// Unit returns the device unit used for new dataset allocation (XSTA UNIT).
func (s *ServerStatus) Unit() (string, error) {
	resp, err := s.xstat("Unit")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// UnixFileType returns the HFS file type (regular file or FIFO) used for transfers (XSTA UNIXFILETYPE).
func (s *ServerStatus) UnixFileType() (string, error) {
	resp, err := s.xstat("UNIXFILETYPE")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// VCount returns the number of volumes (volume count) for a new allocation (XSTA VCOUNT).
func (s *ServerStatus) VCount() (int, error) {
	resp, err := s.xstat("VCOUNT")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

// Volume returns the volume serial used for new dataset allocation (XSTA VOLUME).
func (s *ServerStatus) Volume() (string, error) {
	resp, err := s.xstat("VOLume")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

// WrapRecord reports whether data is wrapped to the next record instead of truncated (XSTA WRAPRECORD).
func (s *ServerStatus) WrapRecord() (string, error) {
	resp, err := s.xstat("WRAPrecord")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// WRTapeFastIo reports whether fast (BSAM) I/O is used when writing tape (XSTA WRTAPEFASTIO).
func (s *ServerStatus) WRTapeFastIo() (string, error) {
	resp, err := s.xstat("WRTAPEFastio")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

// XLate returns the translate-table name used for the data connection (XSTA XLATE).
func (s *ServerStatus) XLate() (string, error) {
	resp, err := s.xstat("XLate")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}
