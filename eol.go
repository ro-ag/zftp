package zftp

import "fmt"

// SetRetrieveEOL sets the end-of-line sequence for the FTP server.
func (s *FTPSession) SetRetrieveEOL(eol LineBreaker) error {
	cmd := fmt.Sprintf("SBSENDEOL=%s", eol.String())
	_, err := s.Site(cmd)
	return err
}

// SetRetrieveWideCharEOL sets the end-of-line wide characters sequence for the FTP server.
func (s *FTPSession) SetRetrieveWideCharEOL(eol LineBreaker) error {
	cmd := fmt.Sprintf("MBSENDEOL=%s", eol.String())
	_, err := s.Site(cmd)
	return err
}

// LineBreaker is an interface for end-of-line sequences.
type LineBreaker interface {
	String() string  // String returns the end-of-line command identifier.
	NewLine() string // NewLine returns the end-of-line sequence.
}

type crlf string

// EolCrlf is the Windows end-of-line sequence.
const EolCrlf = crlf("CRLF")

func (c crlf) String() string {
	return string(c)
}

func (c crlf) NewLine() string {
	return "\r\n"
}

type lf string

// EolLf is the Unix end-of-line sequence.
const EolLf = lf("LF")

func (l lf) String() string {
	return string(l)
}

func (l lf) NewLine() string {
	return "\n"
}

type cr string

// EolCr is the legacy Mac/OS end-of-line sequence. (deprecated)
const EolCr = cr("CR")

func (c cr) String() string {
	return string(c)
}

func (c cr) NewLine() string {
	return "\r"
}
