package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gopkg.in/ro-ag/zftp.v1/internal/log"
	"gopkg.in/ro-ag/zftp.v1/internal/text"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

var RegexSearchPattern = regexp.MustCompile(`[*?]|^\s*$`)

func IsMigrated(line []byte) bool {
	p := Prefix(line)
	return bytes.HasPrefix(bytes.ToLower(p), []byte("migrated"))
}

func IsNotMounted(line []byte) bool {
	p := Prefix(line)
	return bytes.HasPrefix(bytes.ToLower(p), []byte("not mounted"))
}

func Prefix(line []byte) []byte {
	f := bytes.Fields(line)
	return f[0]
}

func PrefixString(line []byte) string {
	return string(Prefix(line))
}

func Suffix(line []byte) []byte {
	f := bytes.Fields(line)
	return f[len(f)-1]
}

func SuffixString(line []byte) string {
	return string(Suffix(line))
}

func Caller() string {
	pc, _, _, _ := runtime.Caller(2)
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}
	return fn.Name()
}

var indentLength = len("INFO") + len("\t")

func WrapText(str string) string {
	lines := strings.Split(str, "\n")

	if len(lines) <= 1 {
		return str // No need to wrap if there is only one line
	}

	var builder strings.Builder
	builder.WriteString(lines[0])
	builder.WriteString("\n")

	wrapped := text.Indent(text.Wrap(strings.Join(lines[1:], "\n"), 80-indentLength), strings.Repeat(" ", indentLength))

	builder.WriteString(wrapped)

	return builder.String()
}

// MaxUint32 is the maximum value that can be represented by a 32-bit unsigned integer.
const MaxUint32 = 4294967296

// VerifyGzSize verifies that the size of the transferred file matches the size in the gzip footer.
// This is necessary because the gzip format only supports files up to 2^32 bytes.
func VerifyGzSize(file *os.File, size int64) error {
	// Sanity check
	const footerSize = 4
	footer := make([]byte, footerSize)

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %s", err)
	}

	_, err = file.ReadAt(footer, int64(fileInfo.Size())-footerSize)
	if err != nil {
		return fmt.Errorf("failed to read gzip footer: %s", err)
	}

	// gzip's footer contains the original file's size modulo 2^32 in the last four bytes
	have := binary.LittleEndian.Uint32(footer)
	want := uint32(size % MaxUint32)

	log.Debugf("file %s size=%d, uncompress size: %d, expected size: %d", file.Name(), fileInfo.Size(), have, want)

	if have != want {
		return fmt.Errorf("transferred file size doesn't match the size in gzip footer (expected %d, got %d)", have, want)
	}
	return nil
}

var trims = regexp.MustCompile(`(^\s+|\s+$|^'|'$)|[\n\r]+`)

func StandardizeQuote(name string) string {
	trims.ReplaceAllString(name, "")
	return fmt.Sprintf("'%s'", name)
}

func RemoveNewLine(name string) string {
	return trims.ReplaceAllString(name, "")
}

var regexLastWord = regexp.MustCompile(`\s(\w+)$`)

func LastWord(str string) string {
	matches := regexLastWord.FindStringSubmatch(str)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func LastWordToInt(str string) (int, error) {
	str = LastWord(str)
	if str == "undefined" {
		return 0, nil
	}
	return strconv.Atoi(str)
}

func LastText(str string) string {
	if len(str) == 0 {
		return ""
	}
	words := strings.Split(RemoveNewLine(str), " ")
	return words[len(words)-1]
}

func StringToBool(str string) (bool, error) {
	switch strings.TrimSpace(strings.ToLower(str)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", str)
	}
}

func LastWordToBool(str string) (bool, error) {
	return StringToBool(LastWord(str))
}

type SetRestorer struct {
	orig string
	want string
	set  func(string) error
	get  func() (string, error)
}

func SetValueAndGetCurrent(value string, set func(string) error, get func() (string, error)) (*SetRestorer, error) {
	orig, err := get()
	if err != nil {
		return nil, fmt.Errorf("failed to get current value: %s", err)
	}

	err = set(value)
	if err != nil {
		return nil, fmt.Errorf("failed to set value: %s", err)
	}

	return &SetRestorer{
		orig: orig,
		set:  set,
		get:  get,
	}, nil
}

func (f *SetRestorer) Restore() {
	err := f.set(f.orig)
	if err != nil {
		log.Warning(err)
	}
}
