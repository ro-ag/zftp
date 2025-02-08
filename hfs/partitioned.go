package hfs

import (
	"fmt"
	"reflect"
	"strings"
)

// Partitioned
// Name     VV.MM   Created       Changed      Size  Init   Mod   Id

// InfoPdsMember represents a member of a Partitioned Dataset
type InfoPdsMember struct {
	Name    FieldString `json:"Name"`    // Name Partitioned InfoDataset Member Name
	VvMm    FieldFloat  `json:"VV.MM"`   // VvMm Version number and modification level. The version number is set to 1 and the modification level is set to 0 when the member is created.
	Created FieldDate   `json:"Created"` // Created The Date this version was created
	Changed FieldTime   `json:"Changed"` // Changed Date and time this version was last modified
	Size    FieldInt    `json:"Size"`    // Size - Number of lines
	Init    FieldInt    `json:"Init"`    // Init Number of lines when the member was first saved
	Mod     FieldInt    `json:"Mod"`     // Mod Number of lines in the current member that have been added or changed. If the data is unnumbered, this number is zero
	Id      FieldString `json:"Id"`      // Id The user ID of the person who created or last updated this version
}

// String returns a row of text representing the Partitioned Dataset member
func (m *InfoPdsMember) String() string {
	str := strings.Builder{}
	str.WriteString(fmt.Sprintf("Name: %s, ", m.Name.String()))
	str.WriteString(fmt.Sprintf("VV.MM: %s, ", m.VvMm.String()))
	str.WriteString(fmt.Sprintf("Created: %s, ", m.Created.String()))
	str.WriteString(fmt.Sprintf("Changed: %s, ", m.Changed.String()))
	str.WriteString(fmt.Sprintf("Size: %s, ", m.Size.String()))
	str.WriteString(fmt.Sprintf("Init: %s, ", m.Init.String()))
	str.WriteString(fmt.Sprintf("Mod: %s, ", m.Mod.String()))
	str.WriteString(fmt.Sprintf("ID: %s", m.Id.String()))
	return str.String()
}

func (m *InfoPdsMember) ToStringSlice() []string {
	return []string{
		m.Name.String(),
		m.VvMm.String(),
		m.Created.String(),
		m.Changed.String(),
		m.Size.String(),
		m.Init.String(),
		m.Mod.String(),
		m.Id.String(),
	}
}

// Headers returns the headers for the dataset
func (m *InfoPdsMember) Headers() []string {
	t := reflect.TypeOf(*m)
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

// ParseInfoPdsMember parses a Partitioned Dataset member recordß
func ParseInfoPdsMember(record string) (InfoPdsMember, error) {
	if len(record) < idOffset {
		return InfoPdsMember{}, fmt.Errorf("record too short: %d", len(record))
	}

	member := InfoPdsMember{}

	err := member.Name.parse(record[nameOffset : nameOffset+nameSize])
	if err != nil {
		return InfoPdsMember{}, fmt.Errorf("failed to parse Name field: %w", err)
	}

	err = member.VvMm.parse(record[vvMmOffset : vvMmOffset+vvMmSize])
	if err != nil {
		return InfoPdsMember{}, fmt.Errorf("failed to parse VvMm field: %w", err)
	}

	err = member.Created.parse(record[createdOffset : createdOffset+createdSize])
	if err != nil {
		return InfoPdsMember{}, fmt.Errorf("failed to parse Created field: %w", err)
	}

	err = member.Changed.parse(record[changedOffset : changedOffset+changedSize])
	if err != nil {
		return InfoPdsMember{}, fmt.Errorf("failed to parse Changed field: %w", err)
	}

	err = member.Size.parse(record[sizeOffset : sizeOffset+sizeSize])
	if err != nil {
		return InfoPdsMember{}, fmt.Errorf("failed to parse Size field: %w", err)
	}

	err = member.Init.parse(record[initOffset : initOffset+initSize])
	if err != nil {
		return InfoPdsMember{}, fmt.Errorf("failed to parse Init field: %w", err)
	}

	err = member.Mod.parse(record[modOffset : modOffset+modSize])
	if err != nil {
		return InfoPdsMember{}, fmt.Errorf("failed to parse Mod field: %w", err)
	}

	err = member.Id.parse(record[idOffset:])
	if err != nil {
		return InfoPdsMember{}, fmt.Errorf("failed to parse Id field: %w", err)
	}

	return member, nil
}
