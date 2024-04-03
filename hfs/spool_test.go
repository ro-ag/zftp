package hfs_test

import (
	"gopkg.in/ro-ag/zftp.v1/hfs"
	"os"
	"strings"
	"testing"
)

func TestParseJobStatus(t *testing.T) {
	t.Run("JesInterfaceLevel=2", func(t *testing.T) {
		bytes, err := os.ReadFile("job_level2_test.txt")
		if err != nil {
			t.Fatal(err)
		}

		lines := strings.Split(string(bytes), "\n")

		fields, err := hfs.ParseInfoJob(lines)

		if err != nil {
			t.Fatal(err)
		}

		for _, field := range fields {
			t.Logf("%+v", field)
		}
	})

	t.Run("JesInterfaceLevel=1", func(t *testing.T) {
		bytes, err := os.ReadFile("job_level1_test.txt")
		if err != nil {
			t.Fatal(err)
		}

		lines := strings.Split(string(bytes), "\n")

		fields, err := hfs.ParseInfoJob(lines)
		if err != nil {
			t.Fatal(err)
		}

		for _, field := range fields {
			t.Logf("%+v", field)
		}
	})

	t.Run("JesSpool", func(t *testing.T) {
		bytes, err := os.ReadFile("job_spool_test.txt")
		if err != nil {
			t.Fatal(err)
		}

		lines := strings.Split(string(bytes), "\n")

		fields, err := hfs.ParseInfoJobDetail(lines)
		if err != nil {
			t.Fatal(err)
		}

		for _, field := range fields.Detail() {
			t.Logf("%+v", field)
		}
	})

	t.Run("JesSpool_Unknown", func(t *testing.T) {
		bytes, err := os.ReadFile("job_spool_unknown_test.txt")
		if err != nil {
			t.Fatal(err)
		}

		lines := strings.Split(string(bytes), "\n")

		_, err = hfs.ParseInfoJobDetail(lines)
		if err != nil {
			if err != hfs.ErrActiveJob {
				t.Fatal(err)
			}
		}
	})

	t.Run("JesSpool_elapsed", func(t *testing.T) {
		bytes, err := os.ReadFile("job_spool_elapsed.txt")
		if err != nil {
			t.Fatal(err)
		}

		lines := strings.Split(string(bytes), "\n")

		_, err = hfs.ParseInfoJobDetail(lines)
		if err != nil {
			if err != hfs.ErrActiveJob {
				t.Fatal(err)
			}
		}
	})

	t.Run("JesSpool_abend", func(t *testing.T) {
		bytes, err := os.ReadFile("job_spool_abend.txt")
		if err != nil {
			t.Fatal(err)
		}

		lines := strings.Split(string(bytes), "\n")

		detail, err := hfs.ParseInfoJobDetail(lines)
		if err != nil {
			t.Fatal(err)
		}

		rc, err := detail.ReturnCode()
		if err != nil {
			if err != hfs.ErrAbendedJob {
				t.Fatal(err)
			}
			t.Logf("%+v with code %d", err, rc)
		}
	})
}
