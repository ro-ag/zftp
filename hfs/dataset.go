// SPDX-License-Identifier: Apache-2.0

package hfs

import (
	"fmt"
	"reflect"
	"strings"
)

// InfoDataset is a struct that represents a z/OS dataset
type InfoDataset struct {
	Dsname     FieldString `json:"Dsname"`
	Volume     FieldString `json:"Volume"`
	Unit       FieldString `json:"Unit"`
	Referred   FieldDate   `json:"Referred"`
	Ext        FieldInt    `json:"Ext"`
	Used       FieldInt    `json:"Used"`
	Recfm      FieldString `json:"Recfm"`
	Lrecl      FieldInt    `json:"Lrecl"`
	BlkSz      FieldInt    `json:"BlkSz"`
	Dsorg      FieldString `json:"Dsorg"`
	isMigrated bool
	isNotMount bool
}

// Name returns DName but without the quotes
func (d *InfoDataset) Name() string {
	return strings.Trim(d.Dsname.String(), "'")
}

// IsMigrated returns true if the dataset is migrated
func (d *InfoDataset) IsMigrated() bool {
	return d.isMigrated
}

// IsNotMounted returns true if the dataset is not mounted
func (d *InfoDataset) IsNotMounted() bool {
	return d.isNotMount
}

// Active returns true if the dataset is not migrated and not, not mounted
func (d *InfoDataset) Active() bool {
	return !d.IsMigrated() && !d.IsNotMounted()
}

// IsPartitioned returns true if the dataset is partitioned
func (d *InfoDataset) IsPartitioned() bool {
	return d.Dsorg.String() == "PO"
}

// IsSequential returns true if the dataset is sequential
func (d *InfoDataset) IsSequential() bool {
	return d.Dsorg.String() == "PS"
}

// IsVSAM returns true if the dataset is a VSAM cluster. z/OS reports VSAM
// entries with "VSAM" in the Dsorg column (volume/unit may be blank or set),
// so detection keys off Dsorg.
func (d *InfoDataset) IsVSAM() bool {
	return strings.EqualFold(d.Dsorg.String(), "VSAM")
}

// IsTape returns true if the dataset is a tape
func (d *InfoDataset) IsTape() bool {
	return strings.ToLower(d.Unit.String()) == "tape"
}

// ToStringSlice returns a slice of strings representing the dataset
func (d *InfoDataset) ToStringSlice() []string {
	return []string{
		d.Dsname.String(),
		d.Volume.String(),
		d.Unit.String(),
		d.Referred.String(),
		d.Ext.String(),
		d.Used.String(),
		d.Recfm.String(),
		d.Lrecl.String(),
		d.BlkSz.String(),
		d.Dsorg.String(),
	}
}

// String return a row of text representing the dataset
func (d *InfoDataset) String() string {
	str := strings.Builder{}
	str.WriteString(fmt.Sprintf("Name: %s, ", d.Dsname.String()))
	str.WriteString(fmt.Sprintf("Volume: %s, ", d.Volume.String()))
	str.WriteString(fmt.Sprintf("Unit: %s, ", d.Unit.String()))
	str.WriteString(fmt.Sprintf("Referred: %s, ", d.Referred.String()))
	str.WriteString(fmt.Sprintf("Ext: %s, ", d.Ext.String()))
	str.WriteString(fmt.Sprintf("Used: %s, ", d.Used.String()))
	str.WriteString(fmt.Sprintf("Recfm: %s, ", d.Recfm.String()))
	str.WriteString(fmt.Sprintf("Lrecl: %s, ", d.Lrecl.String()))
	str.WriteString(fmt.Sprintf("BlkSz: %s, ", d.BlkSz.String()))
	str.WriteString(fmt.Sprintf("Dsorg: %s", d.Dsorg.String()))
	return str.String()
}

// Headers returns the headers for the dataset
func (d *InfoDataset) Headers() []string {
	t := reflect.TypeOf(*d)
	headers := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			headers = append(headers, jsonTag)
		}
	}
	return headers
}

// dsnameStart is the byte offset at which the (always-present) dataset name
// column begins. A record must reach at least this column to be parseable.
const dsnameStart = 56

// dsnameField is parsed for every record kind, including migrated and
// not-mounted entries that carry no other attributes.
var dsnameField = field{"Dsname", dsnameStart, 0}

// datasetFields is the fixed-width column layout of a z/OS dataset LIST record
// (excluding Dsname, which is handled separately). Offsets and widths are
// derived from real server output; see hfs/dataset_test.txt and the golden
// fixtures in hfs/testdata.
var datasetFields = []field{
	{"Volume", 0, 6},
	{"Unit", 6, 5},
	{"Referred", 11, 13},
	{"Ext", 24, 3},
	{"Used", 27, 5},
	{"Recfm", 32, 6},
	{"Lrecl", 38, 6},
	{"BlkSz", 44, 6},
	{"Dsorg", 51, 5},
}

// datasetKind classifies a raw LIST record before column parsing.
type datasetKind int

const (
	dsNormal datasetKind = iota
	dsMigrated
	dsNotMounted
)

// classifyDataset detects the special non-columnar record states that z/OS
// emits in place of attribute columns.
func classifyDataset(record string) datasetKind {
	trimmed := strings.TrimSpace(record)
	switch {
	case strings.HasPrefix(trimmed, "Migrated"):
		return dsMigrated
	case strings.Contains(trimmed, "Not Mounted"):
		return dsNotMounted
	default:
		return dsNormal
	}
}

// setField routes a raw column value to its typed destination on the dataset.
func (d *InfoDataset) setField(name, raw string) error {
	switch name {
	case "Volume":
		return d.Volume.parse(raw)
	case "Unit":
		return d.Unit.parse(raw)
	case "Referred":
		return d.Referred.parse(raw)
	case "Ext":
		return d.Ext.parse(raw)
	case "Used":
		return d.Used.parse(raw)
	case "Recfm":
		return d.Recfm.parse(raw)
	case "Lrecl":
		return d.Lrecl.parse(raw)
	case "BlkSz":
		return d.BlkSz.parse(raw)
	case "Dsorg":
		return d.Dsorg.parse(raw)
	default:
		return fmt.Errorf("unknown dataset field %q", name)
	}
}

// ParseInfoDataset parses a single dataset record from the z/OS FTP "LIST"
// command output. Migrated and not-mounted datasets carry only a name; their
// volume column is set to the state label and the remaining attributes are left
// zero-valued.
func ParseInfoDataset(record string) (InfoDataset, error) {
	if len(record) < dsnameStart+1 {
		return InfoDataset{}, fmt.Errorf("invalid record size: %d", len(record))
	}

	dataset := InfoDataset{}
	if err := dataset.Dsname.parse(dsnameField.slice(record)); err != nil {
		return InfoDataset{}, fmt.Errorf("failed to parse Dsname field: %w", err)
	}

	switch classifyDataset(record) {
	case dsMigrated:
		dataset.isMigrated = true
		if err := dataset.Volume.parse("Migrated"); err != nil {
			return InfoDataset{}, fmt.Errorf("failed to parse Volume field: %w", err)
		}
		return dataset, nil
	case dsNotMounted:
		dataset.isNotMount = true
		if err := dataset.Volume.parse("Not Mounted"); err != nil {
			return InfoDataset{}, fmt.Errorf("failed to parse Volume field: %w", err)
		}
		return dataset, nil
	}

	for _, f := range datasetFields {
		if err := dataset.setField(f.name, f.slice(record)); err != nil {
			return InfoDataset{}, fmt.Errorf("failed to parse %s field: %w", f.name, err)
		}
	}

	return dataset, nil
}
