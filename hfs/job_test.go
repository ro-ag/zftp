package hfs

import (
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

		fields, err := ParseJobStatus(lines, "")
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

		fields, err := ParseJobStatus(lines, "")
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

		fields, err := ParseJobStatus(lines, "")
		if err != nil {
			t.Fatal(err)
		}

		for _, field := range fields {
			t.Logf("%+v", field)
		}
	})

	t.Run("JesSpool_Unknown", func(t *testing.T) {
		bytes, err := os.ReadFile("job_spool_unknown_test.txt")
		if err != nil {
			t.Fatal(err)
		}

		lines := strings.Split(string(bytes), "\n")

		fields, err := ParseJobStatus(lines, "JOB07530")
		if err != nil {
			t.Fatal(err)
		}

		for _, field := range fields {
			t.Logf("%+v", field)
		}
	})
}
