// SPDX-License-Identifier: Apache-2.0

package hfs_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/ro-ag/zftp.v2/hfs"
)

var update = flag.Bool("update", false, "update golden files in testdata/")

// assertGolden compares got against testdata/<name>, or writes it when -update is set.
// Golden files are the exact-match contract: they pin the precise parsed output of the
// real z/OS fixtures so the table-driven rewrite cannot silently change behavior.
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("updated %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (regenerate with: go test ./hfs -run %s -update)", path, err, t.Name())
	}
	if !bytes.Equal(want, got) {
		t.Errorf("golden mismatch for %s\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
	}
}

func toJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(b, '\n')
}

// ---------------------------------------------------------------------------
// Dataset listing
// ---------------------------------------------------------------------------

// dsSnapshot captures the full observable surface of a parsed dataset record
// (display strings + classification booleans). IsVSAM is intentionally omitted
// here and asserted separately, because the rewrite fixes its detection logic.
type dsSnapshot struct {
	Name       string
	Dsname     string
	Volume     string
	Unit       string
	Referred   string
	Ext        string
	Used       string
	Recfm      string
	Lrecl      string
	BlkSz      string
	Dsorg      string
	Migrated   bool
	NotMounted bool
	Active     bool
	PDS        bool
	Sequential bool
	Tape       bool
}

func snapDataset(d hfs.InfoDataset) dsSnapshot {
	return dsSnapshot{
		Name:       d.Name(),
		Dsname:     d.Dsname.String(),
		Volume:     d.Volume.String(),
		Unit:       d.Unit.String(),
		Referred:   d.Referred.String(),
		Ext:        d.Ext.String(),
		Used:       d.Used.String(),
		Recfm:      d.Recfm.String(),
		Lrecl:      d.Lrecl.String(),
		BlkSz:      d.BlkSz.String(),
		Dsorg:      d.Dsorg.String(),
		Migrated:   d.IsMigrated(),
		NotMounted: d.IsNotMounted(),
		Active:     d.Active(),
		PDS:        d.IsPartitioned(),
		Sequential: d.IsSequential(),
		Tape:       d.IsTape(),
	}
}

func TestGolden_ParseInfoDataset(t *testing.T) {
	f, err := os.Open("dataset_test.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var snaps []dsSnapshot
	sc := bufio.NewScanner(f)
	first := true
	for sc.Scan() {
		if first { // skip column header
			first = false
			continue
		}
		line := sc.Text()
		d, err := modernParser.Parse(line)
		if err != nil {
			t.Fatalf("ParseInfoDataset(%q): %v", line, err)
		}
		snaps = append(snaps, snapDataset(d))
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(snaps) == 0 {
		t.Fatal("no dataset records parsed")
	}
	assertGolden(t, "dataset.golden.json", toJSON(t, snaps))
}

// ---------------------------------------------------------------------------
// PDS member listing
// ---------------------------------------------------------------------------

type pdsSnapshot struct {
	Name    string
	VvMm    string
	Created string
	Changed string
	Size    string
	Init    string
	Mod     string
	Id      string
}

func snapPds(m hfs.InfoPdsMember) pdsSnapshot {
	return pdsSnapshot{
		Name:    m.Name.String(),
		VvMm:    m.VvMm.String(),
		Created: m.Created.String(),
		Changed: m.Changed.String(),
		Size:    m.Size.String(),
		Init:    m.Init.String(),
		Mod:     m.Mod.String(),
		Id:      m.Id.String(),
	}
}

func TestGolden_ParseInfoPdsMember(t *testing.T) {
	f, err := os.Open("partitioned_test.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var snaps []pdsSnapshot
	sc := bufio.NewScanner(f)
	first := true
	for sc.Scan() {
		if first { // skip column header
			first = false
			continue
		}
		line := sc.Text()
		m, err := hfs.ParseInfoPdsMember(line)
		if err != nil {
			t.Fatalf("ParseInfoPdsMember(%q): %v", line, err)
		}
		snaps = append(snaps, snapPds(m))
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(snaps) == 0 {
		t.Fatal("no PDS members parsed")
	}
	assertGolden(t, "partitioned.golden.json", toJSON(t, snaps))
}

// ---------------------------------------------------------------------------
// JES job list (interface level 1 and 2)
// ---------------------------------------------------------------------------

type jobSnapshot struct {
	Name       string
	JobId      string
	Owner      string
	Status     string
	Class      string
	SpoolFiles int
}

func snapJobs(js []hfs.InfoJob) []jobSnapshot {
	out := make([]jobSnapshot, len(js))
	for i := range js {
		out[i] = jobSnapshot{
			Name:       js[i].Name.String(),
			JobId:      js[i].JobId.String(),
			Owner:      js[i].Owner.String(),
			Status:     js[i].Status.String(),
			Class:      js[i].Class.String(),
			SpoolFiles: js[i].SpoolFiles,
		}
	}
	return out
}

func TestGolden_ParseInfoJob_Level2(t *testing.T) {
	b, err := os.ReadFile("job_level2_test.txt")
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := hfs.ParseInfoJob(strings.Split(string(b), "\n"))
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "job_level2.golden.json", toJSON(t, snapJobs(jobs)))
}

func TestGolden_ParseInfoJob_Level1(t *testing.T) {
	b, err := os.ReadFile("job_level1_test.txt")
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := hfs.ParseInfoJob(strings.Split(string(b), "\n"))
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "job_level1.golden.json", toJSON(t, snapJobs(jobs)))
}

// ---------------------------------------------------------------------------
// JES job detail (spool) — explicit assertions, including return-code semantics
// ---------------------------------------------------------------------------

func TestParseInfoJobDetail_Normal(t *testing.T) {
	b, err := os.ReadFile("job_spool_test.txt")
	if err != nil {
		t.Fatal(err)
	}
	jd, err := hfs.ParseInfoJobDetail(strings.Split(string(b), "\n"))
	if err != nil {
		t.Fatal(err)
	}
	job := jd.Job()
	if got := job.Name.String(); got != "ANOTHER" {
		t.Errorf("job name = %q, want ANOTHER", got)
	}
	if got := job.JobId.String(); got != "JOB06184" {
		t.Errorf("job id = %q, want JOB06184", got)
	}
	if got := len(jd.Detail()); got != 4 {
		t.Fatalf("detail count = %d, want 4", got)
	}
	d0 := jd.Detail()[0]
	if d0.Id.Value() != 1 || d0.StepName.String() != "JES2" || d0.DDName.String() != "JESMSGLG" || d0.ByteCount.Value() != 1234 {
		t.Errorf("detail[0] = %+v", d0.String())
	}
	rc, err := jd.ReturnCode()
	if err != nil || rc != 0 {
		t.Errorf("ReturnCode() = (%d, %v), want (0, nil)", rc, err)
	}
}

func TestParseInfoJobDetail_Abend(t *testing.T) {
	b, err := os.ReadFile("job_spool_abend.txt")
	if err != nil {
		t.Fatal(err)
	}
	jd, err := hfs.ParseInfoJobDetail(strings.Split(string(b), "\n"))
	if err != nil {
		t.Fatal(err)
	}
	rc, err := jd.ReturnCode()
	if !errors.Is(err, hfs.ErrAbendedJob) {
		t.Fatalf("ReturnCode() err = %v, want ErrAbendedJob", err)
	}
	if rc != -1 {
		t.Errorf("abend ReturnCode rc = %d, want -1", rc)
	}
	if code, ok := jd.AbendCode(); !ok || code != "S806" {
		t.Errorf("AbendCode() = %q,%v, want S806,true", code, ok)
	}
}

func TestParseInfoJobDetail_Active(t *testing.T) {
	for _, name := range []string{"job_spool_elapsed.txt", "job_spool_unknown_test.txt"} {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		_, err = hfs.ParseInfoJobDetail(strings.Split(string(b), "\n"))
		if !errors.Is(err, hfs.ErrActiveJob) {
			t.Errorf("%s: err = %v, want ErrActiveJob", name, err)
		}
	}
}

// ---------------------------------------------------------------------------
// Negative / malformed inputs
// ---------------------------------------------------------------------------

// TestInfoDataset_IsVSAM verifies VSAM detection against the real fixture. z/OS
// flags VSAM clusters with "VSAM" in the Dsorg column, so every such record must
// report IsVSAM()==true and every PS/PO record must report false. This pins the
// fix for the prior always-false detection bug.
func TestInfoDataset_IsVSAM(t *testing.T) {
	f, err := os.Open("dataset_test.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	first := true
	var vsam, nonVSAM int
	for sc.Scan() {
		if first {
			first = false
			continue
		}
		d, err := modernParser.Parse(sc.Text())
		if err != nil {
			t.Fatal(err)
		}
		switch d.Dsorg.String() {
		case "VSAM":
			if !d.IsVSAM() {
				t.Errorf("IsVSAM()=false for VSAM record %s", d.Name())
			}
			vsam++
		case "PS", "PO":
			if d.IsVSAM() {
				t.Errorf("IsVSAM()=true for %s record %s", d.Dsorg.String(), d.Name())
			}
			nonVSAM++
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if vsam == 0 || nonVSAM == 0 {
		t.Fatalf("fixture coverage too thin: vsam=%d nonVSAM=%d", vsam, nonVSAM)
	}
	t.Logf("verified IsVSAM on %d VSAM and %d non-VSAM records", vsam, nonVSAM)
}

func TestParsers_Malformed(t *testing.T) {
	if _, err := modernParser.Parse("too short"); err == nil {
		t.Error("ParseInfoDataset: want error for short record")
	}
	// A short non-empty record is now a valid name-only member (a member with no
	// ISPF statistics); only a blank record is malformed.
	if _, err := hfs.ParseInfoPdsMember("   "); err == nil {
		t.Error("ParseInfoPdsMember: want error for blank record")
	}
	if m, err := hfs.ParseInfoPdsMember("NOSTATS"); err != nil || m.Name.String() != "NOSTATS" {
		t.Errorf("ParseInfoPdsMember(name-only) = (%q, %v), want (NOSTATS, nil)", m.Name.String(), err)
	}
	if _, err := hfs.ParseInfoJob(nil); err == nil {
		t.Error("ParseInfoJob(nil): want error")
	}
	if _, err := hfs.ParseInfoJob([]string{"", "   "}); err == nil {
		t.Error("ParseInfoJob(blank-only): want error")
	}
}
