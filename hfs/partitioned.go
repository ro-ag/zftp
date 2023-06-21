package hfs

import "fmt"

// Partitioned
// Name     VV.MM   Created       Changed      Size  Init   Mod   Id

// PdsMember represents a member of a Partitioned Dataset
type PdsMember struct {
	Name    StringField `json:"Name"`    // Name Partitioned Dataset Member Name
	VvMm    FloatField  `json:"VV.MM"`   // VvMm Version number and modification level. The version number is set to 1 and the modification level is set to 0 when the member is created.
	Created DateField   `json:"Created"` // Created The Date this version was created
	Changed TimeField   `json:"Changed"` // Changed Date and time this version was last modified
	Size    IntField    `json:"Size"`    // Size - Number of lines
	Init    IntField    `json:"Init"`    // Init Number of lines when the member was first saved
	Mod     IntField    `json:"Mod"`     // Mod Number of lines in the current member that have been added or changed. If the data is unnumbered, this number is zero
	Id      StringField `json:"Id"`      // Id The user ID of the person who created or last updated this version
}

const (
	nameOffset    = 0
	nameSize      = 8
	vvMmOffset    = 8
	vvMmSize      = 7
	createdOffset = 15
	createdSize   = 11
	changedOffset = 26
	changedSize   = 17
	sizeOffset    = 43
	sizeSize      = 6
	initOffset    = 49
	initSize      = 6
	modOffset     = 55
	modSize       = 6
	idOffset      = 61
	idSize        = 9
)

func ParseMember(record string) (PdsMember, error) {
	if len(record) < idOffset {
		return PdsMember{}, fmt.Errorf("record too short: %d", len(record))
	}

	member := PdsMember{}

	err := member.Name.parse(record[nameOffset : nameOffset+nameSize])
	if err != nil {
		return PdsMember{}, fmt.Errorf("failed to parse Name field: %v", err)
	}

	err = member.VvMm.parse(record[vvMmOffset : vvMmOffset+vvMmSize])
	if err != nil {
		return PdsMember{}, fmt.Errorf("failed to parse VvMm field: %v", err)
	}

	err = member.Created.parse(record[createdOffset : createdOffset+createdSize])
	if err != nil {
		return PdsMember{}, fmt.Errorf("failed to parse Created field: %v", err)
	}

	err = member.Changed.parse(record[changedOffset : changedOffset+changedSize])
	if err != nil {
		return PdsMember{}, fmt.Errorf("failed to parse Changed field: %v", err)
	}

	err = member.Size.parse(record[sizeOffset : sizeOffset+sizeSize])
	if err != nil {
		return PdsMember{}, fmt.Errorf("failed to parse Size field: %v", err)
	}

	err = member.Init.parse(record[initOffset : initOffset+initSize])
	if err != nil {
		return PdsMember{}, fmt.Errorf("failed to parse Init field: %v", err)
	}

	err = member.Mod.parse(record[modOffset : modOffset+modSize])
	if err != nil {
		return PdsMember{}, fmt.Errorf("failed to parse Mod field: %v", err)
	}

	err = member.Id.parse(record[idOffset:])
	if err != nil {
		return PdsMember{}, fmt.Errorf("failed to parse Id field: %v", err)
	}

	return member, nil
}
