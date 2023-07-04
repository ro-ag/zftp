package transfer

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
)

type DataTransfer interface {
	Transfer(conn net.Conn) (int64, error)
	Command() string
}

type Store struct {
	src io.Reader
}

func (s *Store) Transfer(conn net.Conn) (int64, error) {
	dest := bufio.NewWriter(conn)
	n, err := io.Copy(dest, s.src)
	if err != nil {
		return 0, err
	}

	err = dest.Flush()
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Store) Command() string {
	return "STOR"
}

func NewStore(src io.Reader) *Store {
	return &Store{src: src}
}

/* ------------------------------------------------------------------------------------------------------------------ */

type Retrieve struct {
	dest io.Writer
}

func (r *Retrieve) Transfer(conn net.Conn) (int64, error) {
	n, err := io.Copy(r.dest, conn)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (r *Retrieve) Command() string {
	return "RETR"
}

func NewRetrieve(dest io.Writer) *Retrieve {
	return &Retrieve{dest: dest}
}

/* ------------------------------------------------------------------------------------------------------------------ */

type StoreAscii struct {
	scanner *bufio.Scanner
}

var crlf = []byte("\r\n")

func (s *StoreAscii) Transfer(conn net.Conn) (int64, error) {

	if s.scanner == nil {
		return 0, errors.New("source scanner not initialized")
	}

	dest := bufio.NewWriter(conn)
	size := int64(0)

	for s.scanner.Scan() {
		line := s.scanner.Text()
		n, err := dest.WriteString(line)
		switch {
		case err != nil:
			return size, err
		case n != len(line):
			return size, io.ErrShortWrite
		default:
			size += int64(n)
		}

		n, err = dest.Write(crlf)
		switch {
		case err != nil:
			return size, err
		case n != len(crlf):
			return size, io.ErrShortWrite
		default:
			size += int64(n)
		}
		size += int64(n)
	}

	if err := s.scanner.Err(); err != nil {
		return size, fmt.Errorf("scan error: %w", err)
	}

	if err := dest.Flush(); err != nil {
		return size, fmt.Errorf("flush error: %w", err)
	}

	return size, nil
}

func (s *StoreAscii) Command() string {
	return "STOR"
}

func NewStoreAscii(src io.Reader) *StoreAscii {
	return &StoreAscii{scanner: bufio.NewScanner(src)}
}
