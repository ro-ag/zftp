package zftp_test

import (
	log "github.com/sirupsen/logrus"
	"testing"
	"zftp"
)

func TestFTPSession_Put(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:      true,
		FullTimestamp:    true,
		DisableTimestamp: true,
		//	PadLevelText:     true,
	})
	s, err := zftp.Open(hostname)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Login(username, password)
	if err != nil {
		t.Fatal(err)
	}

	err = s.SetDataSpecs(zftp.Blksize(2403), zftp.Lrecl(120), zftp.RecfmFB)
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
		zftp.Blksize(2400),
		zftp.Lrecl(120),
		zftp.RecfmFB,
	)

	if err != nil {
		t.Fatal(err)
	}

	err = s.Close()
	if err != nil {
		t.Error(err)
	}
}
