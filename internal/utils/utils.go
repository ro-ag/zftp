package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/kr/text"
	log "github.com/sirupsen/logrus"
	"os"
	"runtime"
	"strings"
)

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

	log.Debugf("[***] file %s size=%d, uncompress size: %d, expected size: %d", file.Name(), fileInfo.Size(), have, want)

	if have != want {
		return fmt.Errorf("transferred file size doesn't match the size in gzip footer (expected %d, got %d)", have, want)
	}
	return nil
}
