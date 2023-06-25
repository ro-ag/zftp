package zftp_test

import (
	"gopkg.in/ro-ag/zftp.v0"
	"testing"
)

func TestFTPSession_StaticStat(t *testing.T) {
	// Create a new FTP session
	s, err := zftp.Open(hostname)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Login to the FTP server
	err = s.Login(username, password)
	if err != nil {
		t.Fatal(err)
	}

	return
}
