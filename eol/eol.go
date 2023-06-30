package eol

// LineBreaker is an interface for end-of-line sequences.
type LineBreaker interface {
	String() string  // String returns the end-of-line command identifier.
	NewLine() string // NewLine returns the end-of-line sequence.
}

type crlf string

// Crlf is the Windows end-of-line sequence.
const Crlf = crlf("CRLF")

func (c crlf) String() string {
	return string(c)
}

func (c crlf) NewLine() string {
	return "\r\n"
}

type lf string

// Lf is the Unix end-of-line sequence.
const Lf = lf("LF")

func (l lf) String() string {
	return string(l)
}

func (l lf) NewLine() string {
	return "\n"
}

type cr string

// Cr is the legacy Mac/OS end-of-line sequence. (deprecated)
const Cr = cr("CR")

func (c cr) String() string {
	return string(c)
}

func (c cr) NewLine() string {
	return "\r"
}
