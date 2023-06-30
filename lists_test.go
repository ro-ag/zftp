package zftp_test

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/ro-ag/zftp.v0"
	"testing"
)

func TestFTPSession_List(t *testing.T) {
	s, err := zftp.Open(hostname)
	if err != nil {
		t.Fatal(err)
	}

	defer s.Close()

	err = s.Login(username, password)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("List", func(t *testing.T) {
		if list, err := s.List("*"); err != nil {
			t.Fatal(err)
		} else {
			for _, f := range list {
				log.Printf("%+v", f)
			}
		}
	})

	t.Run("ListWildCard", func(t *testing.T) {
		if list, err := s.List("'ZXP.*'"); err != nil {
			t.Fatal(err)
		} else {
			for _, f := range list {
				log.Printf("%+v", f)
			}
		}
	})

	t.Run("NList", func(t *testing.T) {
		if list, err := s.NList("'*'"); err != nil {
			t.Fatal(err)
		} else {
			for _, f := range list {
				log.Printf("%+v", f)
			}
		}
	})

	t.Run("ListDatasets", func(t *testing.T) {
		if list, err := s.ListDatasets("*"); err != nil {
			t.Fatal(err)
		} else {
			for _, f := range list {
				log.Printf("%+v", f)
			}
		}
	})

	t.Run("ListPds", func(t *testing.T) {
		if list, err := s.ListPds("'ZXP.PUBLIC.JCL(*)'"); err != nil {
			t.Fatal(err)
		} else {
			for _, f := range list {
				log.Printf("%+v", f)
			}
		}
	})

	t.Run("ListJobs", func(t *testing.T) {
		s.SetStatusOf().FileType("JES")
		s.SetStatusOf().JesJobName("*")
		//s.SetStatusOf().JesOwner("*")
		if list, err := s.List("*"); err != nil {
			t.Fatal(err)
		} else {
			for _, f := range list {
				log.Printf("%+v", f)
			}
		}
	})
}
