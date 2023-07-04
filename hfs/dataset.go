package hfs

import (
	"fmt"
	"strings"
)

// InfoDataset is a struct that represents a z/OS dataset
type InfoDataset struct {
	Volume     FieldString `json:"Volume"`
	Unit       FieldString `json:"Unit"`
	Referred   FieldDate   `json:"Referred"`
	Ext        FieldInt    `json:"Ext"`
	Used       FieldInt    `json:"Used"`
	Recfm      FieldString `json:"Recfm"`
	Lrecl      FieldInt    `json:"Lrecl"`
	BlkSz      FieldInt    `json:"BlkSz"`
	Dsorg      FieldString `json:"Dsorg"`
	Dsname     FieldString `json:"Dsname"`
	isMigrated bool
	isNotMount bool
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

// IsVSAM returns true if the dataset is VSAM
func (d *InfoDataset) IsVSAM() bool {
	return strings.ToLower(d.Volume.String()) == "vsam" || strings.ToLower(d.Volume.String()) == "vsam"
}

// IsTape returns true if the dataset is a tape
func (d *InfoDataset) IsTape() bool {
	return strings.ToLower(d.Unit.String()) == "tape"
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

// Constants for field offsets and sizes
const (
	volumeOffset   = 0
	volumeSize     = 6
	unitOffset     = 6
	unitSize       = 5
	referredOffset = 11
	referredSize   = 13
	extOffset      = 24
	extSize        = 3
	usedOffset     = 27
	usedSize       = 5
	recfmOffset    = 32
	recfmSize      = 6
	lreclOffset    = 38
	lreclSize      = 6
	blkSzOffset    = 44
	blkSzSize      = 6
	dsorgOffset    = 51
	dsorgSize      = 5
	dsnameOffset   = 56
	dsnameSize     = 34
)

// ParseInfoDataset parses a dataset record from the HFS command	"LIST"
func ParseInfoDataset(record string) (InfoDataset, error) {
	if len(record) < dsnameOffset+1 {
		return InfoDataset{}, fmt.Errorf("invalid record size: %d", len(record))
	}
	dataset := InfoDataset{}

	err := dataset.Dsname.parse(record[dsnameOffset:])
	if err != nil {
		return InfoDataset{}, fmt.Errorf("failed to parse Dsname field: %v", err)
	}

	if strings.HasPrefix(strings.TrimSpace(record), "Migrated") {
		dataset.isMigrated = true
		return dataset, nil
	}

	if strings.HasPrefix(strings.TrimSpace(record), "Not Mounted") {
		dataset.isNotMount = true
		return dataset, nil
	}

	err = dataset.Volume.parse(record[volumeOffset : volumeOffset+volumeSize])
	if err != nil {
		return InfoDataset{}, fmt.Errorf("failed to parse Volume field: %v", err)
	}

	err = dataset.Unit.parse(record[unitOffset : unitOffset+unitSize])
	if err != nil {
		return InfoDataset{}, fmt.Errorf("failed to parse Unit field: %v", err)
	}

	err = dataset.Referred.parse(record[referredOffset : referredOffset+referredSize])
	if err != nil {
		return InfoDataset{}, fmt.Errorf("failed to parse Referred field: %v", err)
	}

	err = dataset.Ext.parse(record[extOffset : extOffset+extSize])
	if err != nil {
		return InfoDataset{}, fmt.Errorf("failed to parse Ext field: %v", err)
	}

	err = dataset.Used.parse(record[usedOffset : usedOffset+usedSize])
	if err != nil {
		return InfoDataset{}, fmt.Errorf("failed to parse Used field: %v", err)
	}

	err = dataset.Recfm.parse(record[recfmOffset : recfmOffset+recfmSize])
	if err != nil {
		return InfoDataset{}, fmt.Errorf("failed to parse Recfm field: %v", err)
	}

	err = dataset.Lrecl.parse(record[lreclOffset : lreclOffset+lreclSize])
	if err != nil {
		return InfoDataset{}, fmt.Errorf("failed to parse Lrecl field: %v", err)
	}

	err = dataset.BlkSz.parse(record[blkSzOffset : blkSzOffset+blkSzSize])
	if err != nil {
		return InfoDataset{}, fmt.Errorf("failed to parse BlkSz field: %v", err)
	}

	err = dataset.Dsorg.parse(record[dsorgOffset : dsorgOffset+dsorgSize])
	if err != nil {
		return InfoDataset{}, fmt.Errorf("failed to parse Dsorg field: %v", err)
	}

	return dataset, nil
}
