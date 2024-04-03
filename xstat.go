package zftp

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v1/internal/helper"
	"strings"
)

// Stat returns the server status string
func (s *FTPSession) Stat(a ...string) (string, error) {
	return s.SendCommand(CodeSysStatus, "STAT", a...)
}

// XStat - XSTA retrieve individual status variables or properties from the server's current status.
func (s *FTPSession) XStat(feature string) (string, error) {
	out, err := s.SendCommand(CodeSysStatus, "XSTA", fmt.Sprintf("(%s", feature))
	if err != nil {
		return "", err
	}

	out = strings.ReplaceAll(out, "*** end of status ***", "")
	out = strings.TrimSpace(out)
	return out, nil
}

// StatusOf returns a StatusOf interface for the current FTP session.
func (s *FTPSession) StatusOf() StatusOf {
	return helper.GetFeature(s.XStat)
}

// StatusOf interface with methods to retrieve individual status variables or properties from the server's current status.
// https://www.ibm.com/docs/en/zos/2.2.0?topic=fs-status-subcommand-retrieve-status-information-from-remote-host
type StatusOf interface {
	// ASATrans Indicates that the FTP server interprets characters in the first column of ASA files being transferred as print control characters.
	ASATrans() (string, error)

	// AutoMount Indicates automatic mounting of volumes for data sets that are on unmounted volumes.
	AutoMount() (string, error)

	// AutoRecall Indicates automatic recall of migrated data sets.
	AutoRecall() (string, error)

	// BLocks Indicates that primary and secondary space allocations are in blocks.
	BLocks() (string, error)

	// BlockSize Indicates the block size of a newly allocated data set.
	BlockSize() (int, error)

	// BufNo Indicates the number of access method buffers that are to be used when data is read from or written to a data set.
	BufNo() (int, error)

	// CheckpointInterval (CHKptint) Indicates the checkpoint interval for the sending site in a file transfer request.
	CheckpointInterval() (int, error)

	// ConditionDisposition (CONDdisp) Indicates the disposition of the data set if a retrieve operation for a new data set ends before all the data is written.
	ConditionDisposition() (string, error)

	// Cylinders Indicates that primary and secondary space allocations are in cylinders.
	Cylinders() (string, error)

	// DataClass Indicates the SMS data class.
	DataClass() (string, error)

	// DataKeepAlive Indicates the number of seconds that TCP/IP waits while the data connection is inactive before sending a keepalive packet to the FTP client. The value 0 indicates that the DATAKEEPALIVE timer is disabled for this session. For active mode data connections, the keepalive timer that is configured in PROFILE.TCPIP controls how often keepalive packets flow on the data connection. For passive mode data connections, FTP suppresses the PROFILE.TCPIP keepalive timer.
	DataKeepAlive() (int, error)

	// DatasetMode Indicates whether DatasetMode or DIRECTORYMode is in effect.
	DatasetMode() (string, error)

	// DB2 Indicates the Db2® subsystem name.
	DB2() (string, error)

	// DoubleByteSubstitution (DBSUB) Indicates whether substitution is allowed for data bytes that cannot be translated in a double-byte character translation.
	DoubleByteSubstitution() (bool, error)

	// DCBDSN Indicates the name of the MVS™ data set to be used as a model for allocating new data sets.
	DCBDSN() (string, error)

	// Destination (DESt) Indicates the Network Job Entry (NJE) destination to which the files are routed when you enter a PUt command.
	Destination() (string, error)

	// Directory Indicates the number of directory blocks to be allocated for the directory of a PDS.
	Directory() (string, error)

	// DirectoryMode Indicates whether DATASetmode or DIRECTORYMode is in effect.
	DirectoryMode() (string, error)

	// DSNType Indicates the data set name type for new physical sequential data sets.
	//   - SYSTEM The DSNTYPE value from the SMS data class is used. If no SMS data class is defined, or if it does not specify the DSNTYPE value, the system DSNTYPE value is used. This is the default value.
	//   - BASIC Allocates physical sequential data sets as physical sequential basic format data sets.
	//   - LARGE Allocates physical sequential data sets as physical sequential large format data sets.
	DSNType() (string, error)

	// DSWaitTime Indicates the number of minutes the FTP server waits for an MVS data set to become available when a local data set is held by another job or process. The value 0 indicates that the FTP server does not wait to obtain a data set when the data set is being held by another job or process.
	DSWaitTime() (int, error)

	// EATTR Indicates whether newly allocated data sets can have extended attributes and whether new data sets can reside in the EAS of an EAV.
	//
	//   - SYSTEM The data set uses the SMS data class EATTR value. If no SMS data class is defined, or if the data class contains no EATTR specification, the data set is allocated with the system default.
	//   - NO     The data set cannot reside in the EAS, and its VTOC entry cannot contain extended attributes.
	//   - OPT    The data set can reside in the EAS, and its VTOC entry can have extended attributes if the volume supports them.
	EATTR() (string, error)

	// Encoding Indicates the encoding type that is used for conversions between code pages for data transfers.
	Encoding() (string, error)

	// FifoIoTime Indicates the length of time the that FTP server waits for a read from a z/OS® UNIX named pipe or write to a z/OS UNIX named pipe to complete.
	FifoIoTime() (int, error)

	// FifoOpenTime Indicates the length of time that the FTP server waits for an open of a z/OS UNIX named pipe to complete.
	FifoOpenTime() (int, error)

	// FileType Indicates the data set file type.
	FileType() (string, error)

	// FTPKeepAlive Indicates the control connection keepalive timer value in seconds.
	FTPKeepAlive() (int, error)

	// InactiveTime Indicates the inactivity timer to a specified number of seconds.
	InactiveTime() (int, error)

	// ISPFStats Indicates that FTP will create or update ISPF Member statistics when PUt, MPut, or Append subcommands are issued.
	ISPFStats() (bool, error)

	// JesEntryLimit Indicates the number of entries that can be displayed concurrently using a LIST or NLST command.
	JesEntryLimit() (int, error)

	// JesGetByDSN Indicates whether the server should retrieve the file from the MVS system and submit it as a batch job when FILETYPE is JES and JESINTERFACELEVEL is 2, or whether the server should retrieve the JES spool file by the data set name.
	JesGetByDSN() (bool, error)

	// JesJobName Indicates that any command (Get, LIST, DIr, or MGet) should be limited to those jobs, started tasks, APPC/MVS, or TSO output that match the specified value.
	JesJobName() (string, error)

	// JesLrecl Indicates the logical record length (LRecl) for the Job Entry System (JES) internal reader at the foreign host.
	JesLrecl() (int, error)

	// JesOwner Indicates that any command (Get, LIST, DIr, or MGet) should be limited to those jobs, started tasks, APPC/MVS, or TSO output which are owned by the user ID specified.
	JesOwner() (string, error)

	// JesRecfm Indicates the record format for the JES internal reader at the foreign host.
	JesRecfm() (string, error)

	// JesStatus Indicates what type of information should be returned on LIST and NLST commands.
	JesStatus() (string, error)

	// ListLevel Indicates which format the FTP server will use when it replies to the LIST command.
	ListLevel() (int, error)

	// ListSubDir Indicates that wildcard searches should apply to the current working directory and should also span its subdirectories.
	ListSubDir() (bool, error)

	// Lrecl Indicates the logical record length (LRecl) of a newly allocated data set.
	Lrecl() (int, error)

	// MBDataConn Indicates the code pages for the file system and for the network transfer that are used when the server does data conversion during a data transfer.
	MBDataConn() (string, error)

	// MBRequireLastEol Indicates whether the FTP server reports an error when a multibyte file or data set is received from the server with no EOL sequence in the last record received.
	MBRequireLastEol() (bool, error)

	// MBSendEol Indicates which end-of-line sequence to use when the ENCODING value is SBCS, the data is ASCII, and data is being sent to the server.
	MBSendEol() (string, error)

	// MgmtClass Indicates the SMS management class as defined for the target host by your organization.
	MgmtClass() (string, error)

	// MigrateVol Indicates the volume ID for migrated data sets if they do not use IBM® storage management systems.
	MigrateVol() (string, error)

	// PDSType Indicates whether the FTP server creates local MVS directories as partitioned data sets or as partitioned data sets extended.
	PDSType() (string, error)

	// Primary Indicates the number of tracks, blocks, or cylinders for the primary allocation.
	Primary() (string, error)

	// QuotesOverride Indicates that a single quotation mark at the beginning and end of a file name should override the current working directory instead of being appended to the current working directory.
	QuotesOverride() (string, error)

	// RDW Indicates that variable record descriptor words (RDWs) are treated as if they are part of the record and are not discarded during FTP transmission of variable format data sets in stream mode.
	RDW() (string, error)

	// ReadTapeFormat Displays information about an input data set on tape.
	ReadTapeFormat() (string, error)

	// Recfm Displays the data set record format.
	Recfm() (string, error)

	// RetPD Indicates the number of days that a newly allocated data set should be retained.
	RetPD() (int, error)

	// SBDataConn Indicates the conversions between file system and network code pages to be used for data transfers.
	SBDataConn() (int, error)

	// SBSendEol Indicates which end-of-line sequence to use when ENCODING is SBCS, the data is ASCII, and data is being sent to the client.
	SBSendEol() (string, error)

	// SBSub Indicates that substitution is allowed for data bytes that cannot be translated in a single-byte-character translation.
	SBSub() (bool, error)

	// SBSubChar Indicates the value that is used for substitution when SBSUB is also specified.
	SBSubChar() (string, error)

	// Secondary Indicates the number of tracks, blocks, or cylinders for the secondary allocation.
	Secondary() (string, error)

	// SPRead Indicates that the output is in spreadsheet format when the file type is SQL.
	SPRead() (string, error)

	// SQLCol Indicates the SQL output file column headings.
	SQLCol() (string, error)

	// StorageClass (STORClass) Indicates the SMS storage class as defined by your organization for the target host.
	StorageClass() (string, error)

	// TlsRfcLevel Indicates the level of RFC 4217, On Securing FTP with TLS, that is supported by the server.
	TlsRfcLevel() (string, error)

	// Tracks Indicates that primary and secondary space allocations are in tracks.
	Tracks() (string, error)

	// TrailingBlanks Indicates whether the FTP server preserves the trailing blanks in a fixed-format data set when the data is sent to a foreign host.
	TrailingBlanks() (string, error)

	// Truncate Indicates that truncation is permitted.
	Truncate() (string, error)

	// UCount Indicates the number of devices to allocate concurrently to support the allocation request.
	UCount() (int, error)

	// UCSHostCS "Universal Character Set Host Code Set" Indicates the EBCDIC code set to be used when converting to and from Unicode.
	UCSHostCS() (string, error)

	// UCSSub Indicates that in Unicode-to-EBCDIC conversion, the EBCDIC substitution character is used to replace any Unicode character that cannot be successfully converted.
	UCSSub() (string, error)

	// UCSTrunc Indicates that in Unicode-to-EBCDIC conversion, EBCDIC data truncation is allowed.
	UCSTrunc() (string, error)

	// UMask Indicates the file mode creation mask.
	UMask() (int, error)

	// UnicodeFileSystemBOM Indicates whether the FTP server will store incoming Unicode files with a byte order mark.
	UnicodeFileSystemBOM() (string, error)

	// Unit Indicates the unit type for allocation of new data sets.
	Unit() (string, error)

	// UnixFileType Indicates whether the FTP server treats files in its z/OS UNIX file system as regular files or as named pipes.
	UnixFileType() (string, error)

	// VCount Indicates the number of tape data set volumes that an allocated data set can span.
	VCount() (int, error)

	// Volume Indicates the volume serial number for allocation of new data sets.
	Volume() (string, error)

	// WrapRecord Indicates that data is wrapped to the next record if no new-line character is encountered before the logical record length of the receiving file is reached.
	WrapRecord() (string, error)

	// WRTapeFastIo Indicates that ASCII stream data that is being written to tape can be written using BSAM I/O.
	WRTapeFastIo() (string, error)

	// XLate Indicates the translating table to be used for the data connection.
	XLate() (string, error)
}
