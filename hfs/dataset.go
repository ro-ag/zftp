package hfs

import (
	"fmt"
	"strings"
)

// Dataset is a struct that represents a z/OS dataset
type Dataset struct {
	Volume     StringField `json:"Volume"`
	Unit       StringField `json:"Unit"`
	Referred   DateField   `json:"Referred"`
	Ext        IntField    `json:"Ext"`
	Used       IntField    `json:"Used"`
	Recfm      StringField `json:"Recfm"`
	Lrecl      IntField    `json:"Lrecl"`
	BlkSz      IntField    `json:"BlkSz"`
	Dsorg      StringField `json:"Dsorg"`
	Dsname     StringField `json:"Dsname"`
	isMigrated bool
	isNotMount bool
}

// IsMigrated returns true if the dataset is migrated
func (d *Dataset) IsMigrated() bool {
	return d.isMigrated
}

// IsNotMounted returns true if the dataset is not mounted
func (d *Dataset) IsNotMounted() bool {
	return d.isNotMount
}

// Active returns true if the dataset is not migrated and not, not mounted
func (d *Dataset) Active() bool {
	return !d.IsMigrated() && !d.IsNotMounted()
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

func ParseDataset(record string) (Dataset, error) {
	if len(record) < dsnameOffset+1 {
		return Dataset{}, fmt.Errorf("invalid record size: %d", len(record))
	}
	dataset := Dataset{}

	err := dataset.Dsname.parse(record[dsnameOffset:])
	if err != nil {
		return Dataset{}, fmt.Errorf("failed to parse Dsname field: %v", err)
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
		return Dataset{}, fmt.Errorf("failed to parse Volume field: %v", err)
	}

	err = dataset.Unit.parse(record[unitOffset : unitOffset+unitSize])
	if err != nil {
		return Dataset{}, fmt.Errorf("failed to parse Unit field: %v", err)
	}

	err = dataset.Referred.parse(record[referredOffset : referredOffset+referredSize])
	if err != nil {
		return Dataset{}, fmt.Errorf("failed to parse Referred field: %v", err)
	}

	err = dataset.Ext.parse(record[extOffset : extOffset+extSize])
	if err != nil {
		return Dataset{}, fmt.Errorf("failed to parse Ext field: %v", err)
	}

	err = dataset.Used.parse(record[usedOffset : usedOffset+usedSize])
	if err != nil {
		return Dataset{}, fmt.Errorf("failed to parse Used field: %v", err)
	}

	err = dataset.Recfm.parse(record[recfmOffset : recfmOffset+recfmSize])
	if err != nil {
		return Dataset{}, fmt.Errorf("failed to parse Recfm field: %v", err)
	}

	err = dataset.Lrecl.parse(record[lreclOffset : lreclOffset+lreclSize])
	if err != nil {
		return Dataset{}, fmt.Errorf("failed to parse Lrecl field: %v", err)
	}

	err = dataset.BlkSz.parse(record[blkSzOffset : blkSzOffset+blkSzSize])
	if err != nil {
		return Dataset{}, fmt.Errorf("failed to parse BlkSz field: %v", err)
	}

	err = dataset.Dsorg.parse(record[dsorgOffset : dsorgOffset+dsorgSize])
	if err != nil {
		return Dataset{}, fmt.Errorf("failed to parse Dsorg field: %v", err)
	}

	return dataset, nil
}
