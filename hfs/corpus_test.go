// SPDX-License-Identifier: Apache-2.0

package hfs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/ro-ag/zftp.v2/hfs"
)

// corpusLines reads a sanitized listing fixture from ../tests and returns its
// data rows (the column-header line and any blank lines removed), mirroring how
// ListDatasets/ListPds skip the header before parsing.
func corpusLines(t *testing.T, rel string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "tests", rel))
	if err != nil {
		t.Fatalf("read corpus %s: %v", rel, err)
	}
	all := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	rows := make([]string, 0, len(all))
	for i, l := range all {
		if i == 0 || strings.TrimSpace(l) == "" {
			continue // skip the column header and blank lines
		}
		rows = append(rows, l)
	}
	if len(rows) == 0 {
		t.Fatalf("corpus %s had no data rows", rel)
	}
	return rows
}

var datasetCorpus = []string{
	"dataset_listings/01_canonical.txt",
	"dataset_listings/02_special_states.txt",
	"dataset_listings/03_overflow_none_quoted.txt",
}

// TestCorpus_DatasetListings_ParseWithoutError feeds every real-world dataset
// listing row through ParseInfoDataset and requires it to parse. A single
// unparseable row would, in ListDatasets, abort the whole listing — so the
// special-state rows (archived, Pseudo Directory, "Error determining
// attributes", …) must not error.
func TestCorpus_DatasetListings_ParseWithoutError(t *testing.T) {
	for _, file := range datasetCorpus {
		for _, line := range corpusLines(t, file) {
			if _, err := hfs.ParseInfoDataset(line); err != nil {
				t.Errorf("%s: ParseInfoDataset(%q) error: %v", file, line, err)
			}
		}
	}
}

// corpusDsSnap is the observable surface of a parsed corpus dataset record,
// pinned by a golden file so a parser change cannot silently alter the output for
// any real-world row.
type corpusDsSnap struct {
	Name     string
	Volume   string
	Unit     string
	Referred string
	Ext      string
	Used     string
	Overflow bool
	Recfm    string
	Lrecl    string
	BlkSz    string
	Dsorg    string
	State    string
	Active   bool
}

func snapCorpusDs(d hfs.InfoDataset) corpusDsSnap {
	return corpusDsSnap{
		Name: d.Name(), Volume: d.Volume.String(), Unit: d.Unit.String(),
		Referred: d.Referred.String(), Ext: d.Ext.String(), Used: d.Used.String(),
		Overflow: d.Used.IsOverflow(), Recfm: d.Recfm.String(), Lrecl: d.Lrecl.String(),
		BlkSz: d.BlkSz.String(), Dsorg: d.Dsorg.String(), State: d.State(), Active: d.Active(),
	}
}

// TestGolden_Corpus_Datasets pins the exact parsed output of every real-world
// dataset row in the corpus.
func TestGolden_Corpus_Datasets(t *testing.T) {
	var snaps []corpusDsSnap
	for _, file := range datasetCorpus {
		for _, line := range corpusLines(t, file) {
			d, err := hfs.ParseInfoDataset(line)
			if err != nil {
				t.Fatalf("%s: parse %q: %v", file, line, err)
			}
			snaps = append(snaps, snapCorpusDs(d))
		}
	}
	assertGolden(t, "corpus_datasets.golden.json", toJSON(t, snaps))
}

// parseListing reads a fixture, builds a DatasetListParser from its header line,
// and parses the data rows, indexing them by dataset name.
func parseListing(t *testing.T, rel string) map[string]hfs.InfoDataset {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "tests", rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	p := hfs.NewDatasetListParser(lines[0])
	out := map[string]hfs.InfoDataset{}
	for _, l := range lines[1:] {
		if strings.TrimSpace(l) == "" {
			continue
		}
		d, err := p.Parse(l)
		if err != nil {
			t.Fatalf("%s: parse %q: %v", rel, l, err)
		}
		out[d.Name()] = d
	}
	return out
}

// TestDatasetListParser_AlternateFormats verifies the header-driven parser reads
// the alternate column geometries that the fixed-width modern parser cannot:
// the legacy "Date" header (mm/dd/yy), the LISTLEVEL 2 wide format (de-overflowed
// track counts), and the Co:Z SFTP "Tracks"-column format.
func TestDatasetListParser_AlternateFormats(t *testing.T) {
	// Legacy "Volume Unit Date …" header, 2-digit year.
	legacy := parseListing(t, "dataset_listings/04_legacy_date.txt")
	if d := legacy["HLQ.PROJ.MIME.README"]; d.Volume.String() != "STG002" || d.Unit.String() != "3380E" ||
		d.Recfm.String() != "VB" || d.Dsorg.String() != "PS" || d.Referred.String() == "" {
		t.Errorf("legacy row mis-parsed: %+v", d.String())
	}
	if d := legacy["HLQ.PROJ.TEST.PDS"]; !d.IsPartitioned() || d.Used.Value() != 7 {
		t.Errorf("legacy PO row mis-parsed: %+v", d.String())
	}

	// LISTLEVEL 2 wide format: Used can be a 9-digit de-overflowed track count.
	l2 := parseListing(t, "dataset_listings/05_listlevel2.txt")
	if d := l2["HLQ.PROJ.BDATA.XBB"]; d.Volume.String() != "TERNBH" || d.Used.Value() != 327674532 ||
		d.Recfm.String() != "VBS" || d.Dsorg.String() != "PS" {
		t.Errorf("listlevel2 wide-Used row mis-parsed: %+v", d.String())
	}
	if _, ok := l2["HLQ.PROJ.ASM"]; !ok {
		t.Errorf("listlevel2 PO-E row missing")
	}

	// Co:Z SFTP format: distinct Tracks column; '?' Used on a load library.
	coz := parseListing(t, "dataset_listings/06_coz_tracks.txt")
	if d := coz["HLQ.PROJ.AFILE.TXT"]; d.Volume.String() != "WORK84" || d.Tracks.Value() != 1 ||
		d.Recfm.String() != "FB" || d.BlkSz.Value() != 27920 || d.Dsorg.String() != "PS" {
		t.Errorf("coz row mis-parsed: %+v", d.String())
	}
	if d := coz["HLQ.PROJ.COZ.LOADLIB"]; d.Tracks.Value() != 30 || d.Recfm.String() != "U" {
		t.Errorf("coz loadlib row mis-parsed (Tracks=%d Recfm=%q)", d.Tracks.Value(), d.Recfm.String())
	}
}

// datasetsByName parses a fixture and indexes the records by dataset name.
func datasetsByName(t *testing.T, file string) map[string]hfs.InfoDataset {
	t.Helper()
	out := map[string]hfs.InfoDataset{}
	for _, line := range corpusLines(t, file) {
		d, err := hfs.ParseInfoDataset(line)
		if err != nil {
			t.Fatalf("%s: parse %q: %v", file, line, err)
		}
		out[d.Name()] = d
	}
	return out
}

// TestCorpus_DatasetStateClassification asserts each special-state row is
// classified with the right State() and predicate, and that the columnar special
// rows (Tape/VSAM/GDG/PATH) still parse as active datasets with the right field.
func TestCorpus_DatasetStateClassification(t *testing.T) {
	ds := datasetsByName(t, "dataset_listings/02_special_states.txt")

	check := func(name, state string, active bool, pred func(hfs.InfoDataset) bool) {
		d, ok := ds[name]
		if !ok {
			t.Fatalf("row %q not found in fixture", name)
		}
		if d.State() != state {
			t.Errorf("%s: State() = %q, want %q", name, d.State(), state)
		}
		if d.Active() != active {
			t.Errorf("%s: Active() = %v, want %v", name, d.Active(), active)
		}
		if pred != nil && !pred(d) {
			t.Errorf("%s: predicate failed (state %q)", name, d.State())
		}
	}

	check("HLQ.PROJ.OLD.FFS2", "Migrated", false, func(d hfs.InfoDataset) bool { return d.IsMigrated() })
	check("HLQ.PROJ.NOTMOUNT.BKP", "Not Mounted", false, func(d hfs.InfoDataset) bool { return d.IsNotMounted() })
	check("HLQ.PROJ.ARCHIVED.UNITTEST", "Archived", false, func(d hfs.InfoDataset) bool { return d.IsArchived() })
	check("HLQ.PROJ.NONDASD.BACKUP1", "Not a DASD device", false, func(d hfs.InfoDataset) bool { return d.IsArchived() })
	check("HLQ.PROJ.NOTONVOL", "File not on volume", false, nil)
	check("HLQ.PROJ.ERRATTR.DISTLLIB", "Error determining attributes", false, nil)
	check("HLQ.PROJ.ERRATTR.BKP", "Error determining attributes", false, nil)
	check("ETC", "Pseudo Directory", false, func(d hfs.InfoDataset) bool { return d.IsPseudoDirectory() })
	check("HLQ.UCAT", "User catalog connector", false, nil)

	// Columnar special rows remain active datasets with their distinguishing field.
	check("HLQ.PROJ.TAPE.SVCDUMP", "", true, func(d hfs.InfoDataset) bool { return d.IsTape() })
	check("HLQ.PROJ.DDIR", "", true, func(d hfs.InfoDataset) bool { return d.IsVSAM() })
	check("HLQ.PROJ.DDIR.D", "", true, func(d hfs.InfoDataset) bool { return d.IsVSAM() })
	check("HLQ.PROJ.TEST.GDG", "", true, func(d hfs.InfoDataset) bool { return d.Dsorg.String() == "GDG" })
	check("HLQ.PROJ.CSD.PATHA", "", true, func(d hfs.InfoDataset) bool { return d.Dsorg.String() == "PATH" })
}

// TestCorpus_DatasetEdgeFields asserts the overflow marker, **NONE** referred
// date, and quoted dataset names are handled on real-world rows.
func TestCorpus_DatasetEdgeFields(t *testing.T) {
	ds := datasetsByName(t, "dataset_listings/03_overflow_none_quoted.txt")

	if d := ds["HLQ.PROJ.BIG.XBB"]; !d.Used.IsOverflow() {
		t.Errorf("overflow row: Used.IsOverflow() = false, want true (Used=%q)", d.Used.String())
	}
	// **NONE** referred date → zero/empty Referred but still an active dataset.
	if d, ok := ds["HLQ.PROJ.HCD.MSGLOG"]; !ok || d.Referred.String() != "" || !d.Active() {
		t.Errorf("**NONE** row: Referred=%q active=%v, want empty/active", d.Referred.String(), d.Active())
	}
	// Quoted DSN → Name() strips the quotes.
	if _, ok := ds["HLQ.PROJ.T1.HISPAXZ"]; !ok {
		t.Errorf("quoted DSN not unquoted by Name(); keys=%v", keysOf(ds))
	}
}

func keysOf(m map[string]hfs.InfoDataset) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestCorpus_PdsMembers_ParseWithoutError feeds every PDS member row through
// ParseInfoPdsMember and requires it to parse, including members with no ISPF
// statistics (name-only rows shorter than the full record).
func TestCorpus_PdsMembers_ParseWithoutError(t *testing.T) {
	for _, file := range []string{"pds_members/01_with_stats.txt", "pds_members/02_name_only.txt"} {
		for _, line := range corpusLines(t, file) {
			if _, err := hfs.ParseInfoPdsMember(line); err != nil {
				t.Errorf("%s: ParseInfoPdsMember(%q) error: %v", file, line, err)
			}
		}
	}
}
