// SPDX-License-Identifier: Apache-2.0

package mockzos

import (
	"bufio"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func dial(t *testing.T, s *Server) (net.Conn, *bufio.Reader) {
	t.Helper()
	c, err := net.Dial("tcp", s.Addr())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c, bufio.NewReader(c)
}

func readReply(t *testing.T, r *bufio.Reader, wantCode string) string {
	t.Helper()
	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("read reply: %v", err)
	}
	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, wantCode) {
		t.Fatalf("reply %q: want code %s", line, wantCode)
	}
	return line
}

func send(t *testing.T, c net.Conn, line string) {
	t.Helper()
	if _, err := io.WriteString(c, line+"\r\n"); err != nil {
		t.Fatalf("send %q: %v", line, err)
	}
}

func TestServer_GreetingAndLogin(t *testing.T) {
	s := New(t)
	c, r := dial(t, s)

	readReply(t, r, "220")
	send(t, c, "USER me")
	readReply(t, r, "331")
	send(t, c, "PASS pw")
	readReply(t, r, "230")
	send(t, c, "SYST")
	if syst := readReply(t, r, "215"); !strings.Contains(syst, "MVS") {
		t.Errorf("SYST reply missing MVS: %q", syst)
	}
	send(t, c, "QUIT")
	readReply(t, r, "221")
}

var pasvRe = regexp.MustCompile(`\((\d+),(\d+),(\d+),(\d+),(\d+),(\d+)\)`)

func TestServer_PasvDownload(t *testing.T) {
	s := New(t)
	const payload = "VOL001 line one\r\nVOL002 line two\r\n"
	s.DataFor("LIST", "", payload)

	c, r := dial(t, s)
	readReply(t, r, "220")

	send(t, c, "PASV")
	pasv := readReply(t, r, "227")
	m := pasvRe.FindStringSubmatch(pasv)
	if m == nil {
		t.Fatalf("cannot parse PASV reply: %q", pasv)
	}
	p1, _ := strconv.Atoi(m[5])
	p2, _ := strconv.Atoi(m[6])
	port := p1<<8 + p2

	// The client opens the data connection before sending the transfer command.
	dc, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("dial data: %v", err)
	}
	send(t, c, "LIST USER.*")
	readReply(t, r, "125")

	got, err := io.ReadAll(dc)
	if err != nil {
		t.Fatalf("read data: %v", err)
	}
	if string(got) != payload {
		t.Errorf("data = %q, want %q", got, payload)
	}
	readReply(t, r, "250")
}

func TestServer_Upload(t *testing.T) {
	s := New(t)
	c, r := dial(t, s)
	readReply(t, r, "220")

	send(t, c, "PASV")
	m := pasvRe.FindStringSubmatch(readReply(t, r, "227"))
	p1, _ := strconv.Atoi(m[5])
	p2, _ := strconv.Atoi(m[6])
	dc, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(p1<<8+p2))
	if err != nil {
		t.Fatalf("dial data: %v", err)
	}
	send(t, c, "STOR MY.FILE")
	readReply(t, r, "125")
	if _, err := io.WriteString(dc, "hello mainframe"); err != nil {
		t.Fatalf("write data: %v", err)
	}
	_ = dc.Close()
	readReply(t, r, "250")

	got, ok := s.Stored("MY.FILE")
	if !ok || string(got) != "hello mainframe" {
		t.Errorf("stored = %q (ok=%v), want %q", got, ok, "hello mainframe")
	}
}

func TestServer_ScriptOverride(t *testing.T) {
	s := New(t)
	s.Script("STAT", "211-line one", "211 line two")
	c, r := dial(t, s)
	readReply(t, r, "220")
	send(t, c, "STAT")
	readReply(t, r, "211-")
	readReply(t, r, "211 ")
}
