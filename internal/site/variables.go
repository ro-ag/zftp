package site

import (
	"regexp"
	"strconv"
)

type ErrNotImplemented struct {
	Feature string
}

func (e *ErrNotImplemented) Error() string {
	return "Feature not implemented: " + e.Feature
}

type ErrNotFound struct {
	Feature string
}

func (e *ErrNotFound) Error() string {
	return "Feature not found: " + e.Feature
}

var siteBool = map[string]*regexp.Regexp{
	"TapeReadStream":    regexp.MustCompile(`Server site variable TAPEREADSTREAM is set to (TRUE|FALSE)`),
	"JesTrailingBlanks": regexp.MustCompile(`Server site variable JESTRAILINGBLANKS is set to (TRUE|FALSE)`),
	"MbRequireLastEol":  regexp.MustCompile(`Server site variable MBREQUIRELASTEOL is set to (TRUE|FALSE)`),
	"ExtDBSChinese":     regexp.MustCompile(`Server site variable EXTDBSCHINESE is set to (TRUE|FALSE)`),
	"SBSub":             regexp.MustCompile(`SBSUB is set to (TRUE|FALSE)`),
	"ISPFStats":         regexp.MustCompile(`ISPFSTATS is set to (TRUE|FALSE)`),
	"SMS":               regexp.MustCompile(`SMS is (\w+)`),
}

var siteInt = map[string]*regexp.Regexp{
	"InactivityTimer":   regexp.MustCompile(`Inactivity timer is set to (\d+)`),
	"FTPKeepalive":      regexp.MustCompile(`Timer FTPKEEPALIVE is set to (\d+)`),
	"DataKeepalive":     regexp.MustCompile(`Timer DATAKEEPALIVE is set to (\d+)`),
	"DSWaitTime":        regexp.MustCompile(`Timer DSWAITTIME is set to (\d+)`),
	"DSWaitTimeReply":   regexp.MustCompile(`Server site variable DSWAITTIMEREPLY is set to (\d+)`),
	"FIFOOpenTime":      regexp.MustCompile(`Timer FIFOOPENTIME is set to (\d+)`),
	"FIFOIOTime":        regexp.MustCompile(`Timer FIFOIOTIME is set to (\d+)`),
	"VCount":            regexp.MustCompile(`VCOUNT is (\d+)`),
	"JESLRecl":          regexp.MustCompile(`JESLRECL is (\d+)`),
	"JESInterfaceLevel": regexp.MustCompile(`JESINTERFACELEVEL is (\d+)`),
}

var siteString = map[string]*regexp.Regexp{
	"User":                 regexp.MustCompile(`User: (\w+)`),
	"FileType":             regexp.MustCompile(`FileType (\w+)`),
	"JESRecfm":             regexp.MustCompile(`JESRECFM is (\w+)`),
	"SBCSAscii":            regexp.MustCompile(`Outbound SBCS ASCII data uses (\w+) line terminator`),
	"MBCSAscii":            regexp.MustCompile(`Outbound MBCS ASCII data uses (\w+) line terminator`),
	"UnicodeFilesystemBOM": regexp.MustCompile(`Server site variable UNICODEFILESYSTEMBOM is set to (\w+)`),
	"UnixFileType":         regexp.MustCompile(`Server site variable UNIXFILETYPE is set to (\w+)`),
	"SBSUBChar":            regexp.MustCompile(`SBSUBCHAR is set to (\w+)`),
	"DataSetAllocation":    regexp.MustCompile(`Data sets will be allocated using unit (\w+)`),
}

// SearchBoolValue Function to search for a boolean value in the given text
func SearchBoolValue(response string, siteVariable string) (bool, error) {
	regex, ok := siteBool[siteVariable]
	if !ok {
		return false, &ErrNotImplemented{Feature: siteVariable}
	}
	matches := regex.FindStringSubmatch(response)
	if len(matches) == 2 {
		value := matches[1]
		return value == "TRUE", nil
	}
	return false, &ErrNotFound{Feature: siteVariable}
}

// SearchIntValue Function to search for an integer value in the given text
func SearchIntValue(response string, siteVariable string) (int, error) {
	regex, ok := siteInt[siteVariable]
	if !ok {
		return 0, &ErrNotImplemented{Feature: siteVariable}
	}
	matches := regex.FindStringSubmatch(response)
	if len(matches) == 2 {
		value, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, err
		}
		return value, nil
	}
	return 0, &ErrNotFound{Feature: siteVariable}
}

// SearchStringValue Function to search for a string value in the given text
func SearchStringValue(response string, siteVariable string) (string, error) {
	regex, ok := siteString[siteVariable]
	if !ok {
		return "", &ErrNotImplemented{Feature: siteVariable}
	}
	matches := regex.FindStringSubmatch(response)
	if len(matches) == 2 {
		return matches[1], nil
	}
	return "", &ErrNotFound{Feature: siteVariable}
}
