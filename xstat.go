package zftp

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v0/internal/utils"
	"regexp"
	"strconv"
	"strings"
)

var (
	recFmt = regexp.MustCompile(`^Record\s+format\s+(\w+)\s*,\s*Lrecl:\s*(\d+)\s*,\s*Blocksize:\s*(\d+)`)
)

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
	return xsta(s.XStat)
}

type xsta func(string) (string, error)

// StatusOf interface with methods to retrieve individual status variables or properties from the server's current status.
// https://www.ibm.com/docs/en/zos/2.4.0?topic=ftpdata-summary-ftp-client-server-configuration-statements
type StatusOf interface {
	// ASAtrans Indicates that the FTP server interprets characters in the first column of ASA files being transferred as print control characters.
	ASAtrans() (string, error)

	// AUTOMount Indicates automatic mounting of volumes for data sets that are on unmounted volumes.
	AUTOMount() (string, error)

	// AUTORecall Indicates automatic recall of migrated data sets.
	AUTORecall() (string, error)

	// BLocks Indicates that primary and secondary space allocations are in blocks.
	BLocks() (string, error)

	// BLOCKSIze Indicates the block size of a newly allocated data set.
	BLOCKSIze() (int, error)

	// BUfno Indicates the number of access method buffers that are to be used when data is read from or written to a data set.
	BUfno() (int, error)

	// CHKptint Indicates the checkpoint interval for the sending site in a file transfer request.
	CHKptint() (int, error)

	// CONDdisp Indicates the disposition of the data set if a retrieve operation for a new data set ends before all the data is written.
	CONDdisp() (string, error)

	// CYlinders Indicates that primary and secondary space allocations are in cylinders.
	CYlinders() (string, error)

	// DATAClass Indicates the SMS data class.
	DATAClass() (string, error)

	// DATAKEEPALIVE Indicates the number of seconds that TCP/IP waits while the data connection is inactive before sending a keepalive packet to the FTP client. The value 0 indicates that the DATAKEEPALIVE timer is disabled for this session. For active mode data connections, the keepalive timer that is configured in PROFILE.TCPIP controls how often keepalive packets flow on the data connection. For passive mode data connections, FTP suppresses the PROFILE.TCPIP keepalive timer.
	DATAKEEPALIVE() (string, error)

	// DATASetmode Indicates whether DATASetmode or DIRECTORYMode is in effect.
	DATASetmode() (string, error)

	// DB2 Indicates the Db2® subsystem name.
	DB2() (string, error)

	// DBSUB Indicates whether substitution is allowed for data bytes that cannot be translated in a double-byte character translation.
	DBSUB() (string, error)

	// DCbdsn Indicates the name of the MVS™ data set to be used as a model for allocating new data sets.
	DCbdsn() (string, error)

	// DESt Indicates the Network Job Entry (NJE) destination to which the files are routed when you enter a PUt command.
	DESt() (string, error)

	// Directory Indicates the number of directory blocks to be allocated for the directory of a PDS.
	Directory() (string, error)

	// DIRECTORYMode Indicates whether DATASetmode or DIRECTORYMode is in effect.
	DIRECTORYMode() (string, error)

	// DSNTYPE Indicates the data set name type for new physical sequential data sets.
	//   - SYSTEM The DSNTYPE value from the SMS data class is used. If no SMS data class is defined, or if it does not specify the DSNTYPE value, the system DSNTYPE value is used. This is the default value.
	//   - BASIC Allocates physical sequential data sets as physical sequential basic format data sets.
	//   - LARGE Allocates physical sequential data sets as physical sequential large format data sets.
	DSNTYPE() (string, error)

	// DSWAITTIME Indicates the number of minutes the FTP server waits for an MVS data set to become available when a local data set is held by another job or process. The value 0 indicates that the FTP server does not wait to obtain a data set when the data set is being held by another job or process.
	DSWAITTIME() (string, error)

	// EATTR Indicates whether newly allocated data sets can have extended attributes and whether new data sets can reside in the EAS of an EAV.
	//
	//   - SYSTEM The data set uses the SMS data class EATTR value. If no SMS data class is defined, or if the data class contains no EATTR specification, the data set is allocated with the system default.
	//   - NO     The data set cannot reside in the EAS, and its VTOC entry cannot contain extended attributes.
	//   - OPT    The data set can reside in the EAS, and its VTOC entry can have extended attributes if the volume supports them.
	EATTR() (string, error)

	// ENCODING Indicates the encoding type that is used for conversions between code pages for data transfers.
	ENCODING() (string, error)

	// FIFOIOTIME Indicates the length of time the that FTP server waits for a read from a z/OS® UNIX named pipe or write to a z/OS UNIX named pipe to complete.
	FIFOIOTIME() (string, error)

	// FIFOOPENTIME Indicates the length of time that the FTP server waits for an open of a z/OS UNIX named pipe to complete.
	FIFOOPENTIME() (string, error)

	// FILEtype Indicates the data set file type.
	FILEtype() (string, error)

	// FTpkeepalive Indicates the control connection keepalive timer value in seconds.
	FTpkeepalive() (string, error)

	// INactivetime Indicates the inactivity timer to a specified number of seconds.
	INactivetime() (string, error)

	// ISPFSTATS Indicates that FTP will create or update ISPF Member statistics when PUt, MPut, or APpend subcommands are issued.
	ISPFSTATS() (string, error)

	// JESENTRYLimit Indicates the number of entries that can be displayed concurrently using a LIST or NLST command.
	JESENTRYLimit() (string, error)

	// JESGETBYDSN Indicates whether the server should retrieve the file from the MVS system and submit it as a batch job when FILETYPE is JES and JESINTERFACELEVEL is 2, or whether the server should retrieve the JES spool file by the data set name.
	JESGETBYDSN() (string, error)

	// JESJOBName Indicates that any command (Get, LIST, DIr, or MGet) should be limited to those jobs, started tasks, APPC/MVS, or TSO output that match the specified value.
	JESJOBName() (string, error)

	// JESLrecl Indicates the logical record length (LRecl) for the Job Entry System (JES) internal reader at the foreign host.
	JESLrecl() (string, error)

	// JESOwner Indicates that any command (Get, LIST, DIr, or MGet) should be limited to those jobs, started tasks, APPC/MVS, or TSO output which are owned by the user ID specified.
	JESOwner() (string, error)

	// JESRecfm Indicates the record format for the JES internal reader at the foreign host.
	JESRecfm() (string, error)

	// JESSTatus Indicates what type of information should be returned on LIST and NLST commands.
	JESSTatus() (string, error)

	// LISTLEVEL Indicates which format the FTP server will use when it replies to the LIST command.
	LISTLEVEL() (string, error)

	// LISTSUBdir Indicates that wildcard searches should apply to the current working directory and should also span its subdirectories.
	LISTSUBdir() (string, error)

	// LRecl Indicates the logical record length (LRecl) of a newly allocated data set.
	LRecl() (string, error)

	// MBDATACONN Indicates the code pages for the file system and for the network transfer that are used when the server does data conversion during a data transfer.
	MBDATACONN() (string, error)

	// MBREQUIRELASTEOL Indicates whether the FTP server reports an error when a multibyte file or data set is received from the server with no EOL sequence in the last record received.
	MBREQUIRELASTEOL() (string, error)

	// MBSENDEOL Indicates which end-of-line sequence to use when the ENCODING value is SBCS, the data is ASCII, and data is being sent to the server.
	MBSENDEOL() (string, error)

	// MGmtclass Indicates the SMS management class as defined for the target host by your organization.
	MGmtclass() (string, error)

	// MIGratevol Indicates the volume ID for migrated data sets if they do not use IBM® storage management systems.
	MIGratevol() (string, error)

	// PDSTYPE Indicates whether the FTP server creates local MVS directories as partitioned data sets or as partitioned data sets extended.
	PDSTYPE() (string, error)

	// PRImary Indicates the number of tracks, blocks, or cylinders for the primary allocation.
	PRImary() (string, error)

	// QUOtesoverride Indicates that a single quotation mark at the beginning and end of a file name should override the current working directory instead of being appended to the current working directory.
	QUOtesoverride() (string, error)

	// RDW Indicates that variable record descriptor words (RDWs) are treated as if they are part of the record and are not discarded during FTP transmission of variable format data sets in stream mode.
	RDW() (string, error)

	// READTAPEFormat Displays information about an input data set on tape.
	READTAPEFormat() (string, error)

	// RECfm Displays the data set record format.
	RECfm() (string, error)

	// RETpd Indicates the number of days that a newly allocated data set should be retained.
	RETpd() (string, error)

	// SBDataconn Indicates the conversions between file system and network code pages to be used for data transfers.
	SBDataconn() (string, error)

	// SBSENDEOL Indicates which end-of-line sequence to use when ENCODING is SBCS, the data is ASCII, and data is being sent to the client.
	SBSENDEOL() (string, error)

	// SBSUB Indicates that substitution is allowed for data bytes that cannot be translated in a single-byte-character translation.
	SBSUB() (string, error)

	// SBSUBCHAR Indicates the value that is used for substitution when SBSUB is also specified.
	SBSUBCHAR() (string, error)

	// SECondary Indicates the number of tracks, blocks, or cylinders for the secondary allocation.
	SECondary() (string, error)

	// SPRead Indicates that the output is in spreadsheet format when the file type is SQL.
	SPRead() (string, error)

	// SQLCol Indicates the SQL output file column headings.
	SQLCol() (string, error)

	// STOrclass Indicates the SMS storage class as defined by your organization for the target host.
	STOrclass() (string, error)

	// TLSRFCLEVEL Indicates the level of RFC 4217, On Securing FTP with TLS, that is supported by the server.
	TLSRFCLEVEL() (string, error)

	// TRacks Indicates that primary and secondary space allocations are in tracks.
	TRacks() (string, error)

	// TRAILingblanks Indicates whether the FTP server preserves the trailing blanks in a fixed-format data set when the data is sent to a foreign host.
	TRAILingblanks() (string, error)

	// TRUNcate Indicates that truncation is permitted.
	TRUNcate() (string, error)

	// UCOUNT Indicates the number of devices to allocate concurrently to support the allocation request.
	UCOUNT() (string, error)

	// UCSHOSTCS Indicates the EBCDIC code set to be used when converting to and from Unicode.
	UCSHOSTCS() (string, error)

	// UCSSUB Indicates that in Unicode-to-EBCDIC conversion, the EBCDIC substitution character is used to replace any Unicode character that cannot be successfully converted.
	UCSSUB() (string, error)

	// UCSTRUNC Indicates that in Unicode-to-EBCDIC conversion, EBCDIC data truncation is allowed.
	UCSTRUNC() (string, error)

	// UMask Indicates the file mode creation mask.
	UMask() (string, error)

	// UNICODEFILESYSTEMBOM Indicates whether the FTP server will store incoming Unicode files with a byte order mark.
	UNICODEFILESYSTEMBOM() (string, error)

	// Unit Indicates the unit type for allocation of new data sets.
	Unit() (string, error)

	// UNIXFILETYPE Indicates whether the FTP server treats files in its z/OS UNIX file system as regular files or as named pipes.
	UNIXFILETYPE() (string, error)

	// VCOUNT Indicates the number of tape data set volumes that an allocated data set can span.
	VCOUNT() (string, error)

	// VOLume Indicates the volume serial number for allocation of new data sets.
	VOLume() (string, error)

	// WRAPrecord Indicates that data is wrapped to the next record if no new-line character is encountered before the logical record length of the receiving file is reached.
	WRAPrecord() (string, error)

	// WRTAPEFastio Indicates that ASCII stream data that is being written to tape can be written using BSAM I/O.
	WRTAPEFastio() (string, error)

	// XLate Indicates the translating table to be used for the data connection.
	XLate() (string, error)
}

func (x xsta) ASAtrans() (string, error) {
	resp, err := x("ASAtrans")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) AUTOMount() (string, error) {
	resp, err := x("AUTOMount")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) AUTORecall() (string, error) {
	resp, err := x("AUTORecall")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) BLocks() (string, error) {
	resp, err := x("BLocks")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) BLOCKSIze() (int, error) {
	resp, err := x("BLOCKSIze")
	if err != nil {
		return 0, err
	}

	m := recFmt.FindStringSubmatch(resp)
	if len(m) < 4 {
		return 0, fmt.Errorf("unexpected response: %s", resp)
	}

	return strconv.Atoi(m[3])
}

func (x xsta) BUfno() (int, error) {
	resp, err := x("BUfno")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x xsta) CHKptint() (int, error) {
	resp, err := x("CHKptint")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x xsta) CONDdisp() (string, error) {
	resp, err := x("CONDdisp")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) CYlinders() (string, error) {
	resp, err := x("CYlinders")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) DATAClass() (string, error) {
	resp, err := x("DATAClass")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) DATAKEEPALIVE() (string, error) {
	resp, err := x("DATAKEEPALIVE")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) DATASetmode() (string, error) {
	resp, err := x("DATASetmode")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) DB2() (string, error) {
	resp, err := x("DB2")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) DBSUB() (string, error) {
	resp, err := x("DBSUB")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) DCbdsn() (string, error) {
	resp, err := x("DCbdsn")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) DESt() (string, error) {
	resp, err := x("DESt")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) Directory() (string, error) {
	resp, err := x("Directory")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) DIRECTORYMode() (string, error) {
	resp, err := x("DIRECTORYMode")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) DSNTYPE() (string, error) {
	resp, err := x("DSNTYPE")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) DSWAITTIME() (string, error) {
	resp, err := x("DSWAITTIME")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) EATTR() (string, error) {
	resp, err := x("EATTR")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) ENCODING() (string, error) {
	resp, err := x("ENCODING")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) FIFOIOTIME() (string, error) {
	resp, err := x("FIFOIOTIME")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) FIFOOPENTIME() (string, error) {
	resp, err := x("FIFOOPENTIME")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) FILEtype() (string, error) {
	resp, err := x("FILEtype")
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

func (x xsta) FTpkeepalive() (string, error) {
	resp, err := x("FTpkeepalive")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) INactivetime() (string, error) {
	resp, err := x("INactivetime")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) ISPFSTATS() (string, error) {
	resp, err := x("ISPFSTATS")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) JESENTRYLimit() (string, error) {
	resp, err := x("JESENTRYLimit")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) JESGETBYDSN() (string, error) {
	resp, err := x("JESGETBYDSN")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) JESJOBName() (string, error) {
	resp, err := x("JESJOBName")
	if err != nil {
		return "", err
	}
	return utils.LastText(resp), nil
}

func (x xsta) JESLrecl() (string, error) {
	resp, err := x("JESLrecl")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) JESOwner() (string, error) {
	resp, err := x("JESOwner")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) JESRecfm() (string, error) {
	resp, err := x("JESRecfm")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) JESSTatus() (string, error) {
	resp, err := x("JESSTatus")
	if err != nil {
		return "", err
	}
	return utils.LastText(resp), nil
}

func (x xsta) LISTLEVEL() (string, error) {
	resp, err := x("LISTLEVEL")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) LISTSUBdir() (string, error) {
	resp, err := x("LISTSUBdir")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) LRecl() (string, error) {
	resp, err := x("LRecl")
	if err != nil {
		return "", err
	}
	m := recFmt.FindStringSubmatch(resp)
	if len(m) < 4 {
		return "", fmt.Errorf("could not parse LRecl: %s", resp)
	}
	return m[2], nil
}

func (x xsta) MBDATACONN() (string, error) {
	resp, err := x("MBDATACONN")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) MBREQUIRELASTEOL() (string, error) {
	resp, err := x("MBREQUIRELASTEOL")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

var eolFmt = regexp.MustCompile(`uses\s+(\w+)\s+line\s+terminator$`)

func (x xsta) MBSENDEOL() (string, error) {
	resp, err := x("MBSENDEOL")
	if err != nil {
		return "", err
	}
	m := eolFmt.FindStringSubmatch(resp)
	if len(m) < 2 {
		return "", fmt.Errorf("could not parse MBSENDEOL")
	}
	return m[1], nil
}

func (x xsta) MGmtclass() (string, error) {
	resp, err := x("MGmtclass")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) MIGratevol() (string, error) {
	resp, err := x("MIGratevol")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) PDSTYPE() (string, error) {
	resp, err := x("PDSTYPE")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) PRImary() (string, error) {
	resp, err := x("PRImary")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) QUOtesoverride() (string, error) {
	resp, err := x("QUOtesoverride")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) RDW() (string, error) {
	resp, err := x("RDW")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) READTAPEFormat() (string, error) {
	resp, err := x("READTAPEFormat")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) RECfm() (string, error) {
	resp, err := x("RECfm")
	if err != nil {
		return "", err
	}
	m := recFmt.FindStringSubmatch(resp)
	if len(m) < 4 {
		return "", fmt.Errorf("could not parse RECfm")
	}
	return m[1], nil
}

func (x xsta) RETpd() (string, error) {
	resp, err := x("RETpd")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) SBDataconn() (string, error) {
	resp, err := x("SBDataconn")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) SBSENDEOL() (string, error) {
	resp, err := x("SBSENDEOL")
	if err != nil {
		return "", err
	}
	m := eolFmt.FindStringSubmatch(resp)
	if len(m) < 2 {
		return "", fmt.Errorf("could not parse SBSENDEOL")
	}
	return m[1], nil
}

func (x xsta) SBSUB() (string, error) {
	resp, err := x("SBSUB")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) SBSUBCHAR() (string, error) {
	resp, err := x("SBSUBCHAR")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) SECondary() (string, error) {
	resp, err := x("SECondary")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) SPRead() (string, error) {
	resp, err := x("SPRead")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) SQLCol() (string, error) {
	resp, err := x("SQLCol")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) STOrclass() (string, error) {
	resp, err := x("STOrclass")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) TLSRFCLEVEL() (string, error) {
	resp, err := x("TLSRFCLEVEL")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) TRacks() (string, error) {
	resp, err := x("TRacks")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) TRAILingblanks() (string, error) {
	resp, err := x("TRAILingblanks")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) TRUNcate() (string, error) {
	resp, err := x("TRUNcate")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) UCOUNT() (string, error) {
	resp, err := x("UCOUNT")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) UCSHOSTCS() (string, error) {
	resp, err := x("UCSHOSTCS")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) UCSSUB() (string, error) {
	resp, err := x("UCSSUB")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) UCSTRUNC() (string, error) {
	resp, err := x("UCSTRUNC")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) UMask() (string, error) {
	resp, err := x("UMask")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) UNICODEFILESYSTEMBOM() (string, error) {
	resp, err := x("UNICODEFILESYSTEMBOM")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) Unit() (string, error) {
	resp, err := x("Unit")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) UNIXFILETYPE() (string, error) {
	resp, err := x("UNIXFILETYPE")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) VCOUNT() (string, error) {
	resp, err := x("VCOUNT")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) VOLume() (string, error) {
	resp, err := x("VOLume")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x xsta) WRAPrecord() (string, error) {
	resp, err := x("WRAPrecord")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) WRTAPEFastio() (string, error) {
	resp, err := x("WRTAPEFastio")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x xsta) XLate() (string, error) {
	resp, err := x("XLate")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}
