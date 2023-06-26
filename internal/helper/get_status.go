// package helper provides a set of funcitons that are exposed as Interface.
// these are not core functions, just wrappers around the core ones

package helper

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v0/internal/utils"
	"regexp"
	"strconv"
)

var (
	recFmt = regexp.MustCompile(`^Record\s+format\s+(\w+)\s*,\s*Lrecl:\s*(\d+)\s*,\s*Blocksize:\s*(\d+)`)
)

type GetFeature func(string) (string, error)

func (x GetFeature) ASAtrans() (string, error) {
	resp, err := x("ASAtrans")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) AUTOMount() (string, error) {
	resp, err := x("AUTOMount")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) AUTORecall() (string, error) {
	resp, err := x("AUTORecall")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) BLocks() (string, error) {
	resp, err := x("BLocks")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) BLOCKSIze() (int, error) {
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

func (x GetFeature) BUfno() (int, error) {
	resp, err := x("BUfno")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) CHKptint() (int, error) {
	resp, err := x("CHKptint")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) CONDdisp() (string, error) {
	resp, err := x("CONDdisp")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) CYlinders() (string, error) {
	resp, err := x("CYlinders")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) DATAClass() (string, error) {
	resp, err := x("DATAClass")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) DATAKEEPALIVE() (int, error) {
	resp, err := x("DATAKEEPALIVE")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) DATASetmode() (string, error) {
	resp, err := x("DATASetmode")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) DB2() (string, error) {
	resp, err := x("DB2")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) DBSUB() (bool, error) {
	resp, err := x("DBSUB")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

func (x GetFeature) DCbdsn() (string, error) {
	resp, err := x("DCbdsn")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) DESt() (string, error) {
	resp, err := x("DESt")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) Directory() (string, error) {
	resp, err := x("Directory")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) DIRECTORYMode() (string, error) {
	resp, err := x("DIRECTORYMode")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) DSNTYPE() (string, error) {
	resp, err := x("DSNTYPE")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) DSWAITTIME() (int, error) {
	resp, err := x("DSWAITTIME")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) EATTR() (string, error) {
	resp, err := x("EATTR")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) ENCODING() (string, error) {
	resp, err := x("ENCODING")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) FIFOIOTIME() (int, error) {
	resp, err := x("FIFOIOTIME")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) FIFOOPENTIME() (int, error) {
	resp, err := x("FIFOOPENTIME")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) FILEtype() (string, error) {
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

func (x GetFeature) FTpkeepalive() (int, error) {
	resp, err := x("FTpkeepalive")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) INactivetime() (int, error) {
	resp, err := x("INactivetime")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) ISPFSTATS() (bool, error) {
	resp, err := x("ISPFSTATS")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

func (x GetFeature) JESENTRYLimit() (int, error) {
	resp, err := x("JESENTRYLimit")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) JESGETBYDSN() (bool, error) {
	resp, err := x("JESGETBYDSN")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

func (x GetFeature) JESJOBName() (string, error) {
	resp, err := x("JESJOBName")
	if err != nil {
		return "", err
	}
	return utils.LastText(resp), nil
}

func (x GetFeature) JESLrecl() (int, error) {
	resp, err := x("JESLrecl")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) JESOwner() (string, error) {
	resp, err := x("JESOwner")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) JESRecfm() (string, error) {
	resp, err := x("JESRecfm")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) JESSTatus() (string, error) {
	resp, err := x("JESSTatus")
	if err != nil {
		return "", err
	}
	return utils.LastText(resp), nil
}

func (x GetFeature) LISTLEVEL() (int, error) {
	resp, err := x("LISTLEVEL")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) LISTSUBdir() (bool, error) {
	resp, err := x("LISTSUBdir")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

func (x GetFeature) LRecl() (int, error) {
	resp, err := x("LRecl")
	if err != nil {
		return 0, err
	}
	m := recFmt.FindStringSubmatch(resp)
	if len(m) < 4 {
		return 0, fmt.Errorf("could not parse LRecl: %s", resp)
	}
	return strconv.Atoi(m[3])
}

func (x GetFeature) MBDATACONN() (string, error) {
	resp, err := x("MBDATACONN")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) MBREQUIRELASTEOL() (bool, error) {
	resp, err := x("MBREQUIRELASTEOL")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

var eolFmt = regexp.MustCompile(`uses\s+(\w+)\s+line\s+terminator$`)

func (x GetFeature) MBSENDEOL() (string, error) {
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

func (x GetFeature) MGmtclass() (string, error) {
	resp, err := x("MGmtclass")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) MIGratevol() (string, error) {
	resp, err := x("MIGratevol")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) PDSTYPE() (string, error) {
	resp, err := x("PDSTYPE")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) PRImary() (string, error) {
	resp, err := x("PRImary")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) QUOtesoverride() (string, error) {
	resp, err := x("QUOtesoverride")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) RDW() (string, error) {
	resp, err := x("RDW")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) READTAPEFormat() (string, error) {
	resp, err := x("READTAPEFormat")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) RECfm() (string, error) {
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

func (x GetFeature) RETpd() (int, error) {
	resp, err := x("RETpd")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) SBDataconn() (int, error) {
	resp, err := x("SBDataconn")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) SBSENDEOL() (string, error) {
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

func (x GetFeature) SBSUB() (bool, error) {
	resp, err := x("SBSUB")
	if err != nil {
		return false, err
	}
	return utils.LastWordToBool(resp)
}

func (x GetFeature) SBSUBCHAR() (string, error) {
	resp, err := x("SBSUBCHAR")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) SECondary() (string, error) {
	resp, err := x("SECondary")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) SPRead() (string, error) {
	resp, err := x("SPRead")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) SQLCol() (string, error) {
	resp, err := x("SQLCol")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) STOrclass() (string, error) {
	resp, err := x("STOrclass")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) TLSRFCLEVEL() (string, error) {
	resp, err := x("TLSRFCLEVEL")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) TRacks() (string, error) {
	resp, err := x("TRacks")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) TRAILingblanks() (string, error) {
	resp, err := x("TRAILingblanks")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) TRUNcate() (string, error) {
	resp, err := x("TRUNcate")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) UCOUNT() (int, error) {
	resp, err := x("UCOUNT")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) UCSHOSTCS() (string, error) {
	resp, err := x("UCSHOSTCS")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) UCSSUB() (string, error) {
	resp, err := x("UCSSUB")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) UCSTRUNC() (string, error) {
	resp, err := x("UCSTRUNC")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) UMask() (int, error) {
	resp, err := x("UMask")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) UNICODEFILESYSTEMBOM() (string, error) {
	resp, err := x("UNICODEFILESYSTEMBOM")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) Unit() (string, error) {
	resp, err := x("Unit")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) UNIXFILETYPE() (string, error) {
	resp, err := x("UNIXFILETYPE")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) VCOUNT() (int, error) {
	resp, err := x("VCOUNT")
	if err != nil {
		return 0, err
	}
	return utils.LastWordToInt(resp)
}

func (x GetFeature) VOLume() (string, error) {
	resp, err := x("VOLume")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}

func (x GetFeature) WRAPrecord() (string, error) {
	resp, err := x("WRAPrecord")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) WRTAPEFastio() (string, error) {
	resp, err := x("WRTAPEFastio")
	if err != nil {
		return "", err
	}
	return utils.RemoveNewLine(resp), nil
}

func (x GetFeature) XLate() (string, error) {
	resp, err := x("XLate")
	if err != nil {
		return "", err
	}
	return utils.LastWord(resp), nil
}
