package hfs_test

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"gopkg.in/ro-ag/zftp.v1/hfs"
	"os"
	"testing"
)

type Members []hfs.InfoPdsMember

func TestParseMember(t *testing.T) {
	f, err := os.Open("partitioned_test.txt")
	if err != nil {
		t.Fatal(err)
	}
	t.Run("ParseInfoPdsMember", func(t *testing.T) {
		s := bufio.NewScanner(f)
		i := 0
		for s.Scan() {
			if i == 0 {
				i++
				continue
			}
			line := s.Text()
			fields, err := hfs.ParseInfoPdsMember(line)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%4.d %+v", i, fields)
			t.Log(i, fields.Name.String(), fields.VvMm.String(), fields.Changed.String(), fields.Created.String())
			i++
		}
	})
	t.Run("Json", func(t *testing.T) {
		_, err = f.Seek(0, 0)
		if err != nil {
			t.Fatal(err)
		}
		var (
			array = make(Members, 0)
			s     = bufio.NewScanner(f)
			i     = 0
		)
		for s.Scan() {
			if i == 0 {
				i++
				continue
			}
			line := s.Text()
			fields, err := hfs.ParseInfoPdsMember(line)
			if err != nil {
				t.Fatal(err)
			}
			array = append(array, fields)

			i++
		}
		b, err := json.MarshalIndent(array, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		t.Log(string(b))
	})
	t.Run("csv", func(t *testing.T) {
		_, err = f.Seek(0, 0)
		if err != nil {
			t.Fatal(err)
		}
		var (
			array = make(Members, 0)
			s     = bufio.NewScanner(f)
			i     = 0
		)
		for s.Scan() {
			if i == 0 {
				i++
				continue
			}
			line := s.Text()
			fields, err := hfs.ParseInfoPdsMember(line)
			if err != nil {
				t.Fatalf("error %v\n%s", err, line)
			}
			array = append(array, fields)
			i++
		}

		csvWriter := csv.NewWriter(os.Stdout)

		err = csvWriter.Write(array[0].Headers())
		if err != nil {
			t.Fatal(err)
		}

		for i := range array {
			err = csvWriter.Write(array[i].ToStringSlice())
			if err != nil {
				t.Fatal(err)
			}
		}
		csvWriter.Flush()
	})
}
