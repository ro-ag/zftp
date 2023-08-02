package zftp_test

import (
	"gopkg.in/ro-ag/zftp.v0"
	"testing"
)

func TestFTPSession_Put(t *testing.T) {

	s, err := zftp.Open(hostname)
	if err != nil {
		t.Fatal(err)
	}

	s.SetVerbose(zftp.LogAll)

	err = s.Login(username, password)
	if err != nil {
		t.Fatal(err)
	}

	err = s.SetDataSpecs(zftp.WithBlkSize(2403), zftp.WithLrecl(120), zftp.WithRecfmFB)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Put("sample_data.bin", "SAMPDATA.EBCDIC", zftp.TypeBinary)
	if err != nil {
		t.Fatal(err)
	}

	// Put also supports a variadic list of attributes
	err = s.Put("sample_data.txt",
		"SAMPDATA.TXT",
		zftp.TypeAscii,
		zftp.WithBlkSize(2400),
		zftp.WithLrecl(120),
		zftp.WithRecfmFB,
	)

	if err != nil {
		t.Fatal(err)
	}

	err = s.Close()
	if err != nil {
		t.Error(err)
	}
}
