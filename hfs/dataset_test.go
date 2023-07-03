package hfs_test

import (
	"bufio"
	"gopkg.in/ro-ag/zftp.v0/hfs"
	"os"
	"testing"
)

func TestParseDataset(t *testing.T) {
	f, err := os.Open("dataset_test.txt")
	if err != nil {
		t.Fatal(err)
	}

	s := bufio.NewScanner(f)
	i := 0
	for s.Scan() {
		if i == 0 {
			i++
			continue
		}
		line := s.Text()
		fields, err := hfs.ParseInfoDataset(line)
		if err != nil {
			t.Fatalf("error %v\n%s", err, line)
		}
		t.Logf("%4.d %+v", i, fields)
		i++
	}
}
