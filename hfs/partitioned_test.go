package hfs

import (
	"bufio"
	"os"
	"testing"
)

func TestParseMember(t *testing.T) {
	f, err := os.Open("partitioned_test.txt")
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
		fields, err := ParseMember(line)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%4.d %+v", i, fields)
		t.Log(i, fields.Name.String(), fields.VvMm.String(), fields.Changed.String(), fields.Created.String())
		i++
	}
}
