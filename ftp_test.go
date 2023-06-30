package zftp_test

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/ro-ag/zftp.v0"
	"os"
	"testing"
)

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:      true,
		FullTimestamp:    true,
		DisableTimestamp: true,
		//	PadLevelText:     true,
	})
}

var (
	hostname = os.Getenv("ZFTP_HOSTNAME")
	username = os.Getenv("ZFTP_USERNAME")
	password = os.Getenv("ZFTP_PASSWORD")
)

func TestOpen(t *testing.T) {

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
