package zftp

import (
	"gopkg.in/ro-ag/zftp.v0/internal/site"
)

// Stat returns the server status string
func (s *FTPSession) Stat(a ...string) (string, error) {
	return s.SendCommand(CodeSysStatus, "STAT", a...)
}

type StaticStatus struct {
	DataKeepalive        int    // Server's data keepalive timer value in seconds.
	DataSetAllocation    string // Server's data set allocation settings.
	DSWaitTime           int    // Server's data set wait time value in seconds.
	DSWaitTimeReply      int    // Server's data set wait time reply value in seconds.
	ExtDBSChinese        bool   // Indicates whether the server supports Chinese characters in double-byte character sets (DBCS).
	FIFOIOTime           int    // Server's FIFO IO time value in seconds.
	FIFOOpenTime         int    // Server's FIFO open time value in seconds.
	FileType             string // Server's default file type.
	FTPKeepalive         int    // Server's FTP keepalive timer value in seconds.
	InactivityTimer      int    // Server's inactivity timer value in seconds.
	ISPFStats            bool   // Indicates whether the server supports ISPF statistics.
	JESInterfaceLevel    int    // Server's JES interface level value.
	JESLRecl             int    // Server's JES record length (LRECL) value in bytes.
	JESRecfm             string // Server's default record format (RECFM) for JES output.
	MBCSAscii            string // Server's MBCS ASCII data transmission settings.
	MbRequireLastEol     bool   // Indicates whether the server requires a last EOL marker.
	SBCSAscii            string // Server's SBCS ASCII data transmission settings.
	SBSub                bool   // Specifies whether substitution is allowed for untranslatable data bytes.
	SBSUBChar            string // Server's SBSUB character setting.
	SMS                  bool   // Specifies whether the server supports Systems Management Server (SMS).
	TapeReadStream       bool   // Indicates whether the server supports tape read stream mode.
	UnicodeFilesystemBOM string // Server's Unicode filesystem settings.
	UnixFileType         string // Server's default file type for UNIX systems.
	User                 string // Username associated with the server session.
	VCount               int    // Server's VCount value.
}

// Status returns the current server status static values.
func (s *FTPSession) Status() (*StaticStatus, error) {
	response, err := s.SendCommand(CodeSysStatus, "STAT")
	if err != nil {
		return nil, err
	}

	stat := &StaticStatus{}
	var searchErr error

	stat.DataKeepalive, searchErr = site.SearchIntValue(response, "DataKeepalive")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.DataSetAllocation, searchErr = site.SearchStringValue(response, "DataSetAllocation")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.DSWaitTime, searchErr = site.SearchIntValue(response, "DSWaitTime")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.DSWaitTimeReply, searchErr = site.SearchIntValue(response, "DSWaitTimeReply")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.ExtDBSChinese, searchErr = site.SearchBoolValue(response, "ExtDBSChinese")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.FIFOIOTime, searchErr = site.SearchIntValue(response, "FIFOIOTime")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.FIFOOpenTime, searchErr = site.SearchIntValue(response, "FIFOOpenTime")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.FileType, searchErr = site.SearchStringValue(response, "FileType")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.FTPKeepalive, searchErr = site.SearchIntValue(response, "FTPKeepalive")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.InactivityTimer, searchErr = site.SearchIntValue(response, "InactivityTimer")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.JESInterfaceLevel, searchErr = site.SearchIntValue(response, "JESInterfaceLevel")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.JESLRecl, searchErr = site.SearchIntValue(response, "JESLRecl")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.JESRecfm, searchErr = site.SearchStringValue(response, "JESRecfm")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.MBCSAscii, searchErr = site.SearchStringValue(response, "MBCSAscii")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.MbRequireLastEol, searchErr = site.SearchBoolValue(response, "MbRequireLastEol")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.SBCSAscii, searchErr = site.SearchStringValue(response, "SBCSAscii")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.SBSub, searchErr = site.SearchBoolValue(response, "SBSub")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.SBSUBChar, searchErr = site.SearchStringValue(response, "SBSUBChar")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.SMS, searchErr = site.SearchBoolValue(response, "SMS")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.TapeReadStream, searchErr = site.SearchBoolValue(response, "TapeReadStream")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.UnicodeFilesystemBOM, searchErr = site.SearchStringValue(response, "UnicodeFilesystemBOM")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.UnixFileType, searchErr = site.SearchStringValue(response, "UnixFileType")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.User, searchErr = site.SearchStringValue(response, "User")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.VCount, searchErr = site.SearchIntValue(response, "VCount")
	if searchErr != nil {
		return nil, searchErr
	}

	stat.ISPFStats, searchErr = site.SearchBoolValue(response, "ISPFStats")
	if searchErr != nil {
		return nil, searchErr
	}

	return stat, nil
}
