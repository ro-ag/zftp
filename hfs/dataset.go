// SPDX-License-Identifier: Apache-2.0

package hfs

import (
	"fmt"
	"reflect"
	"strings"
)

// InfoDataset is a struct that represents a z/OS dataset
type InfoDataset struct {
	// Dsname is the fully qualified dataset name, including the surrounding
	// single quotes as reported by z/OS.
	Dsname FieldString `json:"Dsname"`
	// Volume is the volume serial (VOLSER) on which the dataset resides.
	Volume FieldString `json:"Volume"`
	// Unit is the device unit/type the dataset is allocated on (e.g. "3390").
	Unit FieldString `json:"Unit"`
	// Referred is the date the dataset was last referenced.
	Referred FieldDate `json:"Referred"`
	// Ext is the number of extents the dataset currently occupies.
	Ext FieldInt `json:"Ext"`
	// Used is the amount of space used, in the units reported by the listing
	// (typically tracks).
	Used FieldInt `json:"Used"`
	// Recfm is the record format (e.g. "FB", "VB", "U").
	Recfm FieldString `json:"Recfm"`
	// Lrecl is the logical record length, in bytes.
	Lrecl FieldInt `json:"Lrecl"`
	// BlkSz is the block size, in bytes.
	BlkSz FieldInt `json:"BlkSz"`
	// Dsorg is the dataset organization (e.g. "PO", "PS", "VSAM").
	Dsorg FieldString `json:"Dsorg"`
	// Tracks is the allocated-track count reported only by the Co:Z SFTP listing
	// format (which has a distinct Tracks column); it is zero for the IBM z/OS FTP
	// formats, which do not carry it.
	Tracks FieldInt `json:"Tracks,omitempty"`
	// state is the status label for a record that carries a status phrase in
	// place of the attribute columns (e.g. "Migrated", "Not Mounted",
	// "Pseudo Directory"); it is empty for a normal, fully-attributed dataset.
	state string
}

// Name returns DName but without the quotes
func (d *InfoDataset) Name() string {
	return strings.Trim(d.Dsname.String(), "'")
}

// State returns the status label of a non-attributed record ("Migrated",
// "Not Mounted", "Archived", "Pseudo Directory", "Error determining attributes",
// …), or "" for a normal dataset that carries its attribute columns.
func (d *InfoDataset) State() string {
	return d.state
}

// IsMigrated returns true if the dataset is migrated
func (d *InfoDataset) IsMigrated() bool {
	return d.state == "Migrated"
}

// IsNotMounted returns true if the dataset is not mounted
func (d *InfoDataset) IsNotMounted() bool {
	return d.state == "Not Mounted"
}

// IsArchived returns true if the dataset is archived to a non-DASD device (z/OS
// reports such entries as "Not Direct Access Device" or "Not a DASD device").
func (d *InfoDataset) IsArchived() bool {
	return d.state == "Archived" || d.state == "Not a DASD device"
}

// IsPseudoDirectory returns true if the record is a pseudo-directory entry — a
// single qualifier level z/OS emits under SITE DIRECTORYMODE rather than a real
// dataset.
func (d *InfoDataset) IsPseudoDirectory() bool {
	return d.state == "Pseudo Directory"
}

// Active returns true if the dataset carries real attributes — i.e. it is not in
// a special state (migrated, not mounted, archived, pseudo-directory, …).
func (d *InfoDataset) Active() bool {
	return d.state == ""
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
func (d InfoDataset) String() string {
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
	t := reflect.TypeFor[InfoDataset]()
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

// datasetStateMarkers maps a status phrase z/OS emits in place of the attribute
// columns to the label recorded on the dataset. The marker is matched within the
// attribute area only (the text before the dataset-name column), so a dataset
// name can never trigger a false state. Order matters only for phrases that could
// co-occur; the listed phrases are mutually exclusive in practice.
var datasetStateMarkers = []struct{ marker, label string }{
	{"Migrated", "Migrated"},
	{"Not Mounted", "Not Mounted"},
	{"Not Direct Access Device", "Archived"},
	{"Not a DASD device", "Not a DASD device"},
	{"File not on volume", "File not on volume"},
	{"Error determining attributes", "Error determining attributes"},
	{"Pseudo Directory", "Pseudo Directory"},
	{"User catalog connector", "User catalog connector"},
}

// datasetStateLabel returns the status label found in the given attribute area
// (the part of a record before the dataset name), or "" when none is present.
func datasetStateLabel(area string) string {
	for _, m := range datasetStateMarkers {
		if strings.Contains(area, m.marker) {
			return m.label
		}
	}
	return ""
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
	case "Tracks":
		return d.Tracks.parse(raw)
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
