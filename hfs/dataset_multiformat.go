// SPDX-License-Identifier: Apache-2.0

package hfs

import (
	"fmt"
	"regexp"
	"strings"
)

// A z/OS FTP dataset LIST comes in several column geometries. The default modern
// IBM format ("Volume Unit Referred Ext Used Recfm Lrecl BlkSz Dsorg Dsname",
// dataset name at a fixed column) is parsed by the fixed-width ParseInfoDataset,
// which deterministically splits jammed columns such as the "1+++++" track
// overflow. The alternate geometries — the legacy "Date" header (mm/dd/yy), the
// LISTLEVEL 2 wide format (de-overflowed track counts), and the Co:Z SFTP format
// (a distinct Tracks column, no Unit) — shift the columns, so they are parsed by
// a header-configured, token-and-type-anchored parser instead.
//
// DatasetListParser selects the right strategy from the listing's header line, so
// callers parse a whole listing through one parser built from the header.
type DatasetListParser struct {
	// fixedWidth selects the proven fixed-width ParseInfoDataset for the modern
	// format; the alternate formats use the token parser configured by cols.
	fixedWidth bool
	cols       datasetColumns
}

// datasetColumns describes which optional columns an alternate-format listing
// carries, derived from its header line.
type datasetColumns struct {
	hasUnit   bool // a device-type column (Unit) precedes the date
	hasTracks bool // a Co:Z "Tracks" column precedes Used
}

// NewDatasetListParser builds a parser for a dataset listing from its column
// header line. An empty or unrecognized header falls back to the modern
// fixed-width format.
func NewDatasetListParser(header string) *DatasetListParser {
	h := strings.ToLower(header)
	hasUnit := strings.Contains(h, "unit")
	hasTracks := strings.Contains(h, "tracks")
	hasReferred := strings.Contains(h, "referred")

	// The modern format is Unit + Referred with no Tracks column; only it splits
	// the fixed-width "1+++++" overflow, so route it to the proven parser. Every
	// other recognizable geometry uses the token parser.
	switch {
	case hasUnit && hasReferred && !hasTracks:
		return &DatasetListParser{fixedWidth: true, cols: datasetColumns{hasUnit: true}}
	case hasTracks || !hasUnit || !hasReferred:
		return &DatasetListParser{cols: datasetColumns{hasUnit: hasUnit, hasTracks: hasTracks}}
	default:
		return &DatasetListParser{fixedWidth: true, cols: datasetColumns{hasUnit: true}}
	}
}

// Parse parses one data row of the listing.
func (p *DatasetListParser) Parse(record string) (InfoDataset, error) {
	if p.fixedWidth {
		return ParseInfoDataset(record)
	}
	return parseDatasetTokens(record, p.cols)
}

// dateToken matches a referred/created date column value: yyyy/mm/dd, mm/dd/yy,
// or a **NONE**/***NONE*** placeholder.
var dateToken = regexp.MustCompile(`^(\d{1,4}/\d{1,2}/\d{1,2}|\*{2,3}NONE\*{2,3})$`)

// parseDatasetTokens parses an alternate-format row by whitespace tokens, anchored
// on value types rather than fixed offsets, so it tolerates the differing column
// widths (and wide values) of the legacy, LISTLEVEL 2, and Co:Z geometries.
//
// Field order is Volume [Unit] Date Ext [Tracks] Used Recfm Lrecl BlkSz Dsorg
// Dsname. The dataset name is always the last token and the Dsorg the one before
// it; the date column anchors the split between the leading volume/unit columns
// and the trailing size/format columns.
func parseDatasetTokens(record string, cols datasetColumns) (InfoDataset, error) {
	d := InfoDataset{}

	tokens := strings.Fields(record)
	if len(tokens) < 2 {
		return InfoDataset{}, fmt.Errorf("invalid record: %q", record)
	}
	dsname := tokens[len(tokens)-1]
	attr := tokens[:len(tokens)-1]

	// A status phrase (Migrated, Pseudo Directory, …) replaces the attribute
	// columns; classify it and stop, mirroring the fixed-width path.
	if label := datasetStateLabel(strings.Join(attr, " ")); label != "" {
		if err := d.Dsname.parse(dsname); err != nil {
			return InfoDataset{}, fmt.Errorf("failed to parse Dsname field: %w", err)
		}
		d.state = label
		switch label {
		case "Migrated", "Not Mounted":
			if err := d.Volume.parse(label); err != nil {
				return InfoDataset{}, fmt.Errorf("failed to parse Volume field: %w", err)
			}
		}
		return d, nil
	}

	if err := d.Dsname.parse(dsname); err != nil {
		return InfoDataset{}, fmt.Errorf("failed to parse Dsname field: %w", err)
	}
	if len(attr) == 0 {
		return d, nil
	}

	// Dsorg is always the last attribute token.
	dsorg := attr[len(attr)-1]
	attr = attr[:len(attr)-1]
	if err := d.Dsorg.parse(dsorg); err != nil {
		return InfoDataset{}, fmt.Errorf("failed to parse Dsorg field: %w", err)
	}

	// Locate the date column; it splits the leading and trailing groups.
	di := -1
	for i, t := range attr {
		if dateToken.MatchString(t) {
			di = i
			break
		}
	}

	if di < 0 {
		// No date: a VSAM/GDG/PATH-style row with no attributes. Recover the
		// leading Volume and (if present) Unit; the rest of the columns are absent.
		setLead(&d, attr, cols)
		return d, nil
	}

	if err := setLead(&d, attr[:di], cols); err != nil {
		return InfoDataset{}, err
	}
	if err := d.Referred.parse(attr[di]); err != nil {
		return InfoDataset{}, fmt.Errorf("failed to parse Referred field: %w", err)
	}

	// Trailing group: Ext [Tracks] Used Recfm Lrecl BlkSz.
	names := []string{"Ext"}
	if cols.hasTracks {
		names = append(names, "Tracks")
	}
	names = append(names, "Used", "Recfm", "Lrecl", "BlkSz")
	mid := attr[di+1:]
	if len(mid) != len(names) {
		return InfoDataset{}, fmt.Errorf("unexpected column count after date: got %d (%v), want %d",
			len(mid), mid, len(names))
	}
	for i, name := range names {
		if err := d.setField(name, mid[i]); err != nil {
			return InfoDataset{}, fmt.Errorf("failed to parse %s field: %w", name, err)
		}
	}
	return d, nil
}

// setLead assigns the leading Volume and optional Unit tokens of a row. It returns
// nil even when no leading tokens are present (a blank volume is valid, e.g. a
// VSAM cluster component).
func setLead(d *InfoDataset, lead []string, cols datasetColumns) error {
	if len(lead) == 0 {
		return nil
	}
	if err := d.Volume.parse(lead[0]); err != nil {
		return fmt.Errorf("failed to parse Volume field: %w", err)
	}
	if cols.hasUnit && len(lead) > 1 {
		if err := d.Unit.parse(lead[1]); err != nil {
			return fmt.Errorf("failed to parse Unit field: %w", err)
		}
	}
	return nil
}
