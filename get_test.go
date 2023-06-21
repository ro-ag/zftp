package zftp_test

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/ro-ag/zftp.v0"
	"testing"
)

func TestFTPSession_Get(t *testing.T) {

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

	err = s.Get("'ZXP.PUBLIC.SAMPDATA'", "sample_data.bin", zftp.TypeBinary)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Get("'ZXP.PUBLIC.SAMPDATA'", "sample_data.txt", zftp.TypeAscii)
	if err != nil {
		t.Fatal(err)
	}

	err = s.GetAndGzip("'ZXP.PUBLIC.SAMPDATA'", "sample_data.txt.gz", zftp.TypeAscii)
	if err != nil {
		t.Fatal(err)
	}

	err = s.GetAndGzip("'ZXP.PUBLIC.SAMPDATA'", "sample_data.bin.gz", zftp.TypeBinary)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Close()
	if err != nil {
		t.Error(err)
	}
}
