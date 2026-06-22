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

func (s *ServerStatus) ASATrans() (string, error) {
	resp, err := s.xstat("ASATrans")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) AutoMount() (string, error) {
	resp, err := s.xstat("AUTOMount")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) AutoRecall() (string, error) {
	resp, err := s.xstat("AUTORecall")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) BLocks() (string, error) {
	resp, err := s.xstat("BLocks")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

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

func (s *ServerStatus) BufNo() (int, error) {
	resp, err := s.xstat("BUfno")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (s *ServerStatus) CheckpointInterval() (int, error) {
	resp, err := s.xstat("CHKptint")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (s *ServerStatus) ConditionDisposition() (string, error) {
	resp, err := s.xstat("CONDdisp")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) Cylinders() (string, error) {
	resp, err := s.xstat("CYlinders")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) DataClass() (string, error) {
	resp, err := s.xstat("DATAClass")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) DataKeepAlive() (int, error) {
	resp, err := s.xstat("DATAKEEPALIVE")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (s *ServerStatus) DatasetMode() (string, error) {
	resp, err := s.xstat("DATASetmode")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) DB2() (string, error) {
	resp, err := s.xstat("DB2")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) DoubleByteSubstitution() (bool, error) {
	resp, err := s.xstat("DBSUB")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

func (s *ServerStatus) DCBDSN() (string, error) {
	resp, err := s.xstat("DCBDSN")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) Destination() (string, error) {
	resp, err := s.xstat("DESt")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) Directory() (string, error) {
	resp, err := s.xstat("Directory")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) DirectoryMode() (string, error) {
	resp, err := s.xstat("DIRECTORYMode")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) DSNType() (string, error) {
	resp, err := s.xstat("DSNTYPE")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) DSWaitTime() (int, error) {
	resp, err := s.xstat("DSWAITTIME")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (s *ServerStatus) EATTR() (string, error) {
	resp, err := s.xstat("EATTR")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) Encoding() (string, error) {
	resp, err := s.xstat("ENCODING")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) FifoIoTime() (int, error) {
	resp, err := s.xstat("FIFOIOTIME")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (s *ServerStatus) FifoOpenTime() (int, error) {
	resp, err := s.xstat("FIFOOPENTIME")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

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

func (s *ServerStatus) FTPKeepAlive() (int, error) {
	resp, err := s.xstat("FTpkeepalive")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (s *ServerStatus) InactiveTime() (int, error) {
	resp, err := s.xstat("INactivetime")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (s *ServerStatus) ISPFStats() (bool, error) {
	resp, err := s.xstat("ISPFSTATS")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

func (s *ServerStatus) JesEntryLimit() (int, error) {
	resp, err := s.xstat("JESENTRYLimit")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (s *ServerStatus) JesGetByDSN() (bool, error) {
	resp, err := s.xstat("JESGETBYDSN")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

func (s *ServerStatus) JesJobName() (string, error) {
	resp, err := s.xstat("JESJOBName")
	if err != nil {
		return "", err
	}
	return utils.LastText(resp), nil
}

func (s *ServerStatus) JesLrecl() (int, error) {
	resp, err := s.xstat("JESLrecl")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (s *ServerStatus) JesOwner() (string, error) {
	resp, err := s.xstat("JESOwner")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) JesRecfm() (string, error) {
	resp, err := s.xstat("JESRecfm")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) JesStatus() (string, error) {
	resp, err := s.xstat("JESSTatus")
	if err != nil {
		return "", err
	}
	return utils.LastText(resp), nil
}

func (s *ServerStatus) ListLevel() (int, error) {
	resp, err := s.xstat("LISTLEVEL")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (s *ServerStatus) ListSubDir() (bool, error) {
	resp, err := s.xstat("LISTSUBdir")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

func (s *ServerStatus) Lrecl() (int, error) {
	resp, err := s.xstat("Lrecl")
	if err != nil {
		return 0, err
	}
	m := recFmt.FindStringSubmatch(resp)
	if len(m) < 4 {
		return 0, fmt.Errorf("could not parse Lrecl: %s", resp)
	}
	return strconv.Atoi(m[3])
}

func (s *ServerStatus) MBDataConn() (string, error) {
	resp, err := s.xstat("MBDATACONN")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) MBRequireLastEol() (bool, error) {
	resp, err := s.xstat("MBREQUIRELASTEOL")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

var eolFmt = regexp.MustCompile(`uses\s+(\w+)\s+line\s+terminator$`)

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

func (s *ServerStatus) MgmtClass() (string, error) {
	resp, err := s.xstat("MGmtclass")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) MigrateVol() (string, error) {
	resp, err := s.xstat("MIGratevol")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) PDSType() (string, error) {
	resp, err := s.xstat("PDSTYPE")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) Primary() (string, error) {
	resp, err := s.xstat("Primary")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) QuotesOverride() (string, error) {
	resp, err := s.xstat("QUOtesoverride")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) RDW() (string, error) {
	resp, err := s.xstat("RDW")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) ReadTapeFormat() (string, error) {
	resp, err := s.xstat("READTAPEFormat")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

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

func (s *ServerStatus) RetPD() (int, error) {
	resp, err := s.xstat("RetPD")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (s *ServerStatus) SBDataConn() (int, error) {
	resp, err := s.xstat("SBDataConn")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

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

func (s *ServerStatus) SBSub() (bool, error) {
	resp, err := s.xstat("SBSUB")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

func (s *ServerStatus) SBSubChar() (string, error) {
	resp, err := s.xstat("SBSUBCHAR")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) Secondary() (string, error) {
	resp, err := s.xstat("SECondary")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) SPRead() (string, error) {
	resp, err := s.xstat("SPRead")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) SQLCol() (string, error) {
	resp, err := s.xstat("SQLCol")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) StorageClass() (string, error) {
	resp, err := s.xstat("STOrclass")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) TlsRfcLevel() (string, error) {
	resp, err := s.xstat("TLSRFCLEVEL")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) Tracks() (string, error) {
	resp, err := s.xstat("TRacks")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) TrailingBlanks() (string, error) {
	resp, err := s.xstat("TRAILingblanks")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) Truncate() (string, error) {
	resp, err := s.xstat("TRUNcate")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) UCount() (int, error) {
	resp, err := s.xstat("UCOUNT")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (s *ServerStatus) UCSHostCS() (string, error) {
	resp, err := s.xstat("UCSHOSTCS")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) UCSSub() (string, error) {
	resp, err := s.xstat("UCSSUB")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) UCSTrunc() (string, error) {
	resp, err := s.xstat("UCSTRUNC")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) UMask() (int, error) {
	resp, err := s.xstat("UMask")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (s *ServerStatus) UnicodeFileSystemBOM() (string, error) {
	resp, err := s.xstat("UNICODEFILESYSTEMBOM")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) Unit() (string, error) {
	resp, err := s.xstat("Unit")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) UnixFileType() (string, error) {
	resp, err := s.xstat("UNIXFILETYPE")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) VCount() (int, error) {
	resp, err := s.xstat("VCOUNT")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (s *ServerStatus) Volume() (string, error) {
	resp, err := s.xstat("VOLume")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (s *ServerStatus) WrapRecord() (string, error) {
	resp, err := s.xstat("WRAPrecord")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) WRTapeFastIo() (string, error) {
	resp, err := s.xstat("WRTAPEFastio")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (s *ServerStatus) XLate() (string, error) {
	resp, err := s.xstat("XLate")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}
