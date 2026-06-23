// SPDX-License-Identifier: Apache-2.0

package hfs

import (
	"fmt"
	"strings"
)

// A z/OS FTP dataset LIST comes in several fixed-width column geometries. They
// differ in which columns are present (a Unit device column, a Co:Z Tracks
// column) and in their offsets, but all are positional — which is what lets the
// parser split jammed columns such as the "1+++++" / "165535" Ext+Used overflow
// that carries no gap. Each geometry is described by a datasetLayout; the parser
// slices a row by the layout selected from the listing's header line.
type datasetLayout struct {
	fields      []field
	dsnameStart int
}

var (
	// modernDatasetLayout is the default IBM z/OS FTP format
	// (Volume Unit Referred Ext Used Recfm Lrecl BlkSz Dsorg Dsname); its offsets
	// live in datasetFields/dsnameStart and are pinned by the golden fixtures.
	modernDatasetLayout = datasetLayout{fields: datasetFields, dsnameStart: dsnameStart}

	// legacyDatasetLayout is the older OS/390 format whose date column is labeled
	// "Date" and rendered as a 2-digit-year mm/dd/yy (parsed by FieldDate).
	legacyDatasetLayout = datasetLayout{dsnameStart: 55, fields: []field{
		{"Volume", 0, 6}, {"Unit", 6, 6}, {"Referred", 12, 10}, {"Ext", 22, 4},
		{"Used", 26, 6}, {"Recfm", 32, 6}, {"Lrecl", 38, 6}, {"BlkSz", 44, 6},
		{"Dsorg", 50, 5},
	}}

	// listlevel2DatasetLayout is the SITE LISTLEVEL=2 wide format: no Unit column,
	// and a wide Used column that shows de-overflowed track counts (up to 9 digits).
	listlevel2DatasetLayout = datasetLayout{dsnameStart: 56, fields: []field{
		{"Volume", 0, 6}, {"Referred", 6, 12}, {"Ext", 18, 4}, {"Used", 22, 11},
		{"Recfm", 33, 6}, {"Lrecl", 39, 6}, {"BlkSz", 45, 6}, {"Dsorg", 51, 5},
	}}

	// cozDatasetLayout is the Co:Z SFTP format: no Unit column, and an extra Tracks
	// column between Ext and Used.
	cozDatasetLayout = datasetLayout{dsnameStart: 62, fields: []field{
		{"Volume", 0, 6}, {"Referred", 6, 12}, {"Ext", 18, 4}, {"Tracks", 22, 8},
		{"Used", 30, 8}, {"Recfm", 38, 6}, {"Lrecl", 44, 6}, {"BlkSz", 50, 6},
		{"Dsorg", 56, 6},
	}}
)

// DatasetListParser parses the rows of a dataset LIST according to the column
// geometry detected from the listing's header line, so callers parse a whole
// listing through one parser built from the header.
type DatasetListParser struct {
	layout datasetLayout
}

// NewDatasetListParser builds a parser for a dataset listing from its column
// header line. An empty or unrecognized header falls back to the modern format.
func NewDatasetListParser(header string) *DatasetListParser {
	h := strings.ToLower(header)
	hasUnit := strings.Contains(h, "unit")
	hasReferred := strings.Contains(h, "referred")
	switch {
	case strings.Contains(h, "tracks"):
		return &DatasetListParser{cozDatasetLayout}
	case hasUnit && !hasReferred: // Volume Unit Date …
		return &DatasetListParser{legacyDatasetLayout}
	case !hasUnit && hasReferred: // wide LISTLEVEL 2, no Unit
		return &DatasetListParser{listlevel2DatasetLayout}
	default:
		return &DatasetListParser{modernDatasetLayout}
	}
}

// Parse parses one data row of the listing.
func (p *DatasetListParser) Parse(record string) (InfoDataset, error) {
	return parseDatasetLayout(record, p.layout)
}

// parseDatasetLayout parses a dataset record by slicing the fixed-width columns of
// the given layout. Records that carry a status phrase in place of the attribute
// columns — migrated, not mounted, archived (non-DASD), a pseudo-directory,
// "Error determining attributes", and similar — are classified into State() with
// their attributes left zero-valued rather than being rejected, so one such row
// never aborts a whole listing. Migrated and not-mounted entries additionally
// report their state in the Volume column for backward compatibility.
func parseDatasetLayout(record string, layout datasetLayout) (InfoDataset, error) {
	if len(record) < layout.dsnameStart+1 {
		return InfoDataset{}, fmt.Errorf("invalid record size: %d", len(record))
	}

	d := InfoDataset{}
	if err := d.Dsname.parse(record[layout.dsnameStart:]); err != nil {
		return InfoDataset{}, fmt.Errorf("failed to parse Dsname field: %w", err)
	}

	// The dataset-name column is excluded from the status-phrase search so a name
	// like HLQ.MIGRATED.X cannot be misread as a state.
	if label := datasetStateLabel(record[:layout.dsnameStart]); label != "" {
		d.state = label
		switch label {
		case "Migrated", "Not Mounted":
			if err := d.Volume.parse(label); err != nil {
				return InfoDataset{}, fmt.Errorf("failed to parse Volume field: %w", err)
			}
		}
		return d, nil
	}

	for _, f := range layout.fields {
		if err := d.setField(f.name, f.slice(record)); err != nil {
			return InfoDataset{}, fmt.Errorf("failed to parse %s field: %w", f.name, err)
		}
	}
	return d, nil
}
