// SPDX-License-Identifier: Apache-2.0

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
	t := reflect.TypeFor[InfoPdsMember]()
	headers := make([]string, 0, t.NumField())
	for field := range t.Fields() {
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			headers = append(headers, jsonTag)
		}
	}
	return headers
}

// idStart is the byte offset of the trailing Id column; a record must reach it
// to be considered a parseable member line.
const idStart = 61

// pdsFields is the fixed-width column layout of a z/OS PDS member listing.
// The Id column (width 0) runs to the end of the record.
var pdsFields = []field{
	{"Name", 0, 8},
	{"VvMm", 8, 7},
	{"Created", 15, 11},
	{"Changed", 26, 17},
	{"Size", 43, 6},
	{"Init", 49, 6},
	{"Mod", 55, 6},
	{"Id", idStart, 0},
}

// setField routes a raw column value to its typed destination on the member.
func (m *InfoPdsMember) setField(name, raw string) error {
	switch name {
	case "Name":
		return m.Name.parse(raw)
	case "VvMm":
		return m.VvMm.parse(raw)
	case "Created":
		return m.Created.parse(raw)
	case "Changed":
		return m.Changed.parse(raw)
	case "Size":
		return m.Size.parse(raw)
	case "Init":
		return m.Init.parse(raw)
	case "Mod":
		return m.Mod.parse(raw)
	case "Id":
		return m.Id.parse(raw)
	default:
		return fmt.Errorf("unknown PDS member field %q", name)
	}
}

// ParseInfoPdsMember parses a single Partitioned Dataset member record from the
// z/OS FTP listing output.
//
// A member saved without ISPF statistics is rendered as just its name — a record
// too short to carry the statistics columns. Such a row is parsed into a member
// whose Name is set and whose statistics are left zero-valued, rather than being
// rejected (which would abort the whole listing in ListPds).
func ParseInfoPdsMember(record string) (InfoPdsMember, error) {
	member := InfoPdsMember{}

	if len(record) < idStart {
		name := strings.TrimSpace(record)
		if name == "" {
			return InfoPdsMember{}, fmt.Errorf("empty member record")
		}
		if fields := strings.Fields(name); len(fields) > 0 {
			name = fields[0]
		}
		if err := member.Name.parse(name); err != nil {
			return InfoPdsMember{}, fmt.Errorf("failed to parse Name field: %w", err)
		}
		return member, nil
	}

	for _, f := range pdsFields {
		if err := member.setField(f.name, f.slice(record)); err != nil {
			return InfoPdsMember{}, fmt.Errorf("failed to parse %s field: %w", f.name, err)
		}
	}

	return member, nil
}
