package zftp_test

import (
	"gopkg.in/ro-ag/zftp.v1"
	"os"
	"testing"
)

var (
	hostname = os.Getenv("ZFTP_HOSTNAME")
	username = os.Getenv("ZFTP_USERNAME")
	password = os.Getenv("ZFTP_PASSWORD")
)

func requireEnv(t *testing.T) {
	if hostname == "" || username == "" || password == "" {
		t.Skip("FTP server credentials not configured")
	}
}

func TestOpen(t *testing.T) {
	requireEnv(t)

	s, err := zftp.Open(hostname)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Login(username, password)
	if err != nil {
		t.Fatal(err)
	}

	str, err := s.SendCommand(zftp.CodeHelpMsg, "HELP")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)

	err = s.Close()
	if err != nil {
		t.Fatal(err)
	}
}
