// package helper provides a set of funcitons that are exposed as Interface.
// these are not core functions, just wrappers around the core ones

package helper

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v1/internal/utils"
	"regexp"
	"strconv"
)

var (
	recFmt = regexp.MustCompile(`^Record\s+format\s+(\w+)\s*,\s*Lrecl:\s*(\d+)\s*,\s*Blocksize:\s*(\d+)`)
)

type GetFeature func(string) (string, error)

func (xstat GetFeature) ASATrans() (string, error) {
	resp, err := xstat("ASATrans")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) AutoMount() (string, error) {
	resp, err := xstat("AUTOMount")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) AutoRecall() (string, error) {
	resp, err := xstat("AUTORecall")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) BLocks() (string, error) {
	resp, err := xstat("BLocks")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) BlockSize() (int, error) {
	resp, err := xstat("BLOCKSIze")
	if err != nil {
		return 0, err
	}

	m := recFmt.FindStringSubmatch(resp)
	if len(m) < 4 {
		return 0, fmt.Errorf("unexpected response: %s", resp)
	}

	return strconv.Atoi(m[3])
}

func (xstat GetFeature) BufNo() (int, error) {
	resp, err := xstat("BUfno")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) CheckpointInterval() (int, error) {
	resp, err := xstat("CHKptint")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) ConditionDisposition() (string, error) {
	resp, err := xstat("CONDdisp")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) Cylinders() (string, error) {
	resp, err := xstat("CYlinders")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) DataClass() (string, error) {
	resp, err := xstat("DATAClass")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) DataKeepAlive() (int, error) {
	resp, err := xstat("DATAKEEPALIVE")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) DatasetMode() (string, error) {
	resp, err := xstat("DATASetmode")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) DB2() (string, error) {
	resp, err := xstat("DB2")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) DoubleByteSubstitution() (bool, error) {
	resp, err := xstat("DBSUB")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

func (xstat GetFeature) DCBDSN() (string, error) {
	resp, err := xstat("DCBDSN")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) Destination() (string, error) {
	resp, err := xstat("DESt")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) Directory() (string, error) {
	resp, err := xstat("Directory")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) DirectoryMode() (string, error) {
	resp, err := xstat("DIRECTORYMode")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) DSNType() (string, error) {
	resp, err := xstat("DSNTYPE")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) DSWaitTime() (int, error) {
	resp, err := xstat("DSWAITTIME")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) EATTR() (string, error) {
	resp, err := xstat("EATTR")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) Encoding() (string, error) {
	resp, err := xstat("ENCODING")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) FifoIoTime() (int, error) {
	resp, err := xstat("FIFOIOTIME")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) FifoOpenTime() (int, error) {
	resp, err := xstat("FIFOOPENTIME")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) FileType() (string, error) {
	resp, err := xstat("FileType")
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

func (xstat GetFeature) FTPKeepAlive() (int, error) {
	resp, err := xstat("FTpkeepalive")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) InactiveTime() (int, error) {
	resp, err := xstat("INactivetime")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) ISPFStats() (bool, error) {
	resp, err := xstat("ISPFSTATS")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

func (xstat GetFeature) JesEntryLimit() (int, error) {
	resp, err := xstat("JESENTRYLimit")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) JesGetByDSN() (bool, error) {
	resp, err := xstat("JESGETBYDSN")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

func (xstat GetFeature) JesJobName() (string, error) {
	resp, err := xstat("JESJOBName")
	if err != nil {
		return "", err
	}
	return utils.LastText(resp), nil
}

func (xstat GetFeature) JesLrecl() (int, error) {
	resp, err := xstat("JESLrecl")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) JesOwner() (string, error) {
	resp, err := xstat("JESOwner")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) JesRecfm() (string, error) {
	resp, err := xstat("JESRecfm")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) JesStatus() (string, error) {
	resp, err := xstat("JESSTatus")
	if err != nil {
		return "", err
	}
	return utils.LastText(resp), nil
}

func (xstat GetFeature) ListLevel() (int, error) {
	resp, err := xstat("LISTLEVEL")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) ListSubDir() (bool, error) {
	resp, err := xstat("LISTSUBdir")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

func (xstat GetFeature) Lrecl() (int, error) {
	resp, err := xstat("Lrecl")
	if err != nil {
		return 0, err
	}
	m := recFmt.FindStringSubmatch(resp)
	if len(m) < 4 {
		return 0, fmt.Errorf("could not parse Lrecl: %s", resp)
	}
	return strconv.Atoi(m[3])
}

func (xstat GetFeature) MBDataConn() (string, error) {
	resp, err := xstat("MBDATACONN")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) MBRequireLastEol() (bool, error) {
	resp, err := xstat("MBREQUIRELASTEOL")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

var eolFmt = regexp.MustCompile(`uses\s+(\w+)\s+line\s+terminator$`)

func (xstat GetFeature) MBSendEol() (string, error) {
	resp, err := xstat("MBSENDEOL")
	if err != nil {
		return "", err
	}
	m := eolFmt.FindStringSubmatch(resp)
	if len(m) < 2 {
		return "", fmt.Errorf("could not parse MBSENDEOL")
	}
	return m[1], nil
}

func (xstat GetFeature) MgmtClass() (string, error) {
	resp, err := xstat("MGmtclass")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) MigrateVol() (string, error) {
	resp, err := xstat("MIGratevol")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) PDSType() (string, error) {
	resp, err := xstat("PDSTYPE")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) Primary() (string, error) {
	resp, err := xstat("Primary")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) QuotesOverride() (string, error) {
	resp, err := xstat("QUOtesoverride")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) RDW() (string, error) {
	resp, err := xstat("RDW")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) ReadTapeFormat() (string, error) {
	resp, err := xstat("READTAPEFormat")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) Recfm() (string, error) {
	resp, err := xstat("Recfm")
	if err != nil {
		return "", err
	}
	m := recFmt.FindStringSubmatch(resp)
	if len(m) < 4 {
		return "", fmt.Errorf("could not parse Recfm")
	}
	return m[1], nil
}

func (xstat GetFeature) RetPD() (int, error) {
	resp, err := xstat("RetPD")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) SBDataConn() (int, error) {
	resp, err := xstat("SBDataConn")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) SBSendEol() (string, error) {
	resp, err := xstat("SBSENDEOL")
	if err != nil {
		return "", err
	}
	m := eolFmt.FindStringSubmatch(resp)
	if len(m) < 2 {
		return "", fmt.Errorf("could not parse SBSENDEOL")
	}
	return m[1], nil
}

func (xstat GetFeature) SBSub() (bool, error) {
	resp, err := xstat("SBSUB")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

func (xstat GetFeature) SBSubChar() (string, error) {
	resp, err := xstat("SBSUBCHAR")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) Secondary() (string, error) {
	resp, err := xstat("SECondary")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) SPRead() (string, error) {
	resp, err := xstat("SPRead")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) SQLCol() (string, error) {
	resp, err := xstat("SQLCol")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) StorageClass() (string, error) {
	resp, err := xstat("STOrclass")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) TlsRfcLevel() (string, error) {
	resp, err := xstat("TLSRFCLEVEL")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) Tracks() (string, error) {
	resp, err := xstat("TRacks")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) TrailingBlanks() (string, error) {
	resp, err := xstat("TRAILingblanks")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) Truncate() (string, error) {
	resp, err := xstat("TRUNcate")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) UCount() (int, error) {
	resp, err := xstat("UCOUNT")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) UCSHostCS() (string, error) {
	resp, err := xstat("UCSHOSTCS")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) UCSSub() (string, error) {
	resp, err := xstat("UCSSUB")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) UCSTrunc() (string, error) {
	resp, err := xstat("UCSTRUNC")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) UMask() (int, error) {
	resp, err := xstat("UMask")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) UnicodeFileSystemBOM() (string, error) {
	resp, err := xstat("UNICODEFILESYSTEMBOM")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) Unit() (string, error) {
	resp, err := xstat("Unit")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) UnixFileType() (string, error) {
	resp, err := xstat("UNIXFILETYPE")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) VCount() (int, error) {
	resp, err := xstat("VCOUNT")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (xstat GetFeature) Volume() (string, error) {
	resp, err := xstat("VOLume")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (xstat GetFeature) WrapRecord() (string, error) {
	resp, err := xstat("WRAPrecord")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) WRTapeFastIo() (string, error) {
	resp, err := xstat("WRTAPEFastio")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (xstat GetFeature) XLate() (string, error) {
	resp, err := xstat("XLate")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}
