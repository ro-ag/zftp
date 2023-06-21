package zftp_test

import (
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
	"zftp"
)

var (
	hostname = os.Getenv("ZFTP_HOSTNAME")
	username = os.Getenv("ZFTP_USERNAME")
	password = os.Getenv("ZFTP_PASSWORD")
)

func TestOpenTls(t *testing.T) {

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

	str, err := s.SendCommand(zftp.CodeHelpMsg, "HELP")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)

	lines, err := s.List("'ZXP.*'")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(lines)

	lines, err = s.NList("*")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(lines)

	datasets, err := s.ListDatasets("'ZXP.*'")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(datasets)

	lines, err = s.List("'ZXP.PUBLIC.SOURCE(*)'")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(lines)

	members, err := s.ListPds("'ZXP.PUBLIC.JCL(*)'")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(members)

	err = s.Close()
	if err != nil {
		t.Fatal(err)
	}

}
