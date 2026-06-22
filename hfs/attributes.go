// SPDX-License-Identifier: Apache-2.0

// Package hfs provides tools for interacting with the Hierarchical File System (HFS) on z/OS systems.
// It includes functionalities to manage HFS attributes, handle different types of datasets such as partitioned and sequential datasets,
// and interact with the Job Entry Subsystem (JES) spool. The package provides structured data types to represent jobs and job details,
// and includes functions to parse job records and details from the JES spool.

package hfs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type FieldString struct {
	data string
}

func (f *FieldString) parse(data string) error {
	f.data = strings.TrimSpace(data)
	return nil
}

func (f *FieldString) String() string {
	return f.data
}

func (f *FieldString) Value() string {
	return f.data
}

func (f *FieldString) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.String())
}

func (f *FieldString) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return f.parse(s)
}

/* ------------------------------------------------------------------------------------------------------------------ */

// overflowMarker is the token z/OS prints in a fixed-width numeric column when
// the underlying value is too wide to display (e.g. "+++++"). It is surfaced
// verbatim by String/MarshalJSON so a display overflow can never be mistaken for
// a real number such as 65535.
const overflowMarker = "+++++"

// errFieldIntJSON is returned when decoding a FieldInt from a JSON string that
// is not the recognised overflow marker.
var errFieldIntJSON = errors.New("invalid FieldInt JSON value")

// FieldInt holds a non-negative integer column from a z/OS listing. The value is
// stored as a uint32 so the full width of the source columns (up to six digits)
// is representable without truncation. A z/OS display overflow is recorded out
// of band via overflow rather than by reusing a magic numeric value, so callers
// can always tell a genuine maximum from an undisplayable one (see IsOverflow).
type FieldInt struct {
	data     uint32
	overflow bool
}

func (f *FieldInt) parse(data string) error {
	data = strings.TrimSpace(data)
	f.data = 0
	f.overflow = false
	if len(data) == 0 {
		return nil
	}
	// z/OS fills a numeric column with '+' when the value exceeds the column's
	// display width. That is a display-overflow indicator, not a number: flag it
	// out of band so it stays distinguishable from any real value.
	if strings.ContainsRune(data, '+') {
		f.overflow = true
		return nil
	}
	// ParseUint (base 10, 32-bit) rejects a sign prefix and any value past
	// uint32, so an out-of-range or negative column errors instead of silently
	// wrapping the way uint16(value) used to.
	value, err := strconv.ParseUint(data, 10, 32)
	if err != nil {
		return fmt.Errorf("failed to parse integer field %q: %w", data, err)
	}
	f.data = uint32(value)
	return nil
}

func (f *FieldInt) String() string {
	if f.overflow {
		return overflowMarker
	}
	if f.data == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(f.data), 10)
}

// IsOverflow reports whether the source column held a z/OS display-overflow
// indicator ("+++++") rather than a representable number. When true, Value() is
// 0 and carries no meaning.
func (f *FieldInt) IsOverflow() bool {
	return f.overflow
}

// Value returns the parsed integer. It is 0 for both an absent column and a
// display overflow; use IsOverflow to tell the two apart.
func (f *FieldInt) Value() uint32 {
	return f.data
}

func (f *FieldInt) MarshalJSON() ([]byte, error) {
	if f.overflow {
		return json.Marshal(overflowMarker)
	}
	if f.data == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(f.data)
}

func (f *FieldInt) UnmarshalJSON(b []byte) error {
	f.data = 0
	f.overflow = false
	if string(b) == "null" {
		return nil
	}
	// Overflow is serialised as the marker string; a real value is a JSON number.
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		if s != overflowMarker {
			return fmt.Errorf("%w: %q", errFieldIntJSON, s)
		}
		f.overflow = true
		return nil
	}
	var v uint32
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	f.data = v
	return nil
}

/* ------------------------------------------------------------------------------------------------------------------ */

type FieldFloat struct {
	data float32
}

func (f *FieldFloat) parse(data string) error {
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		f.data = 0.0
		return nil
	}
	value, err := strconv.ParseFloat(data, 32)
	if err != nil {
		return fmt.Errorf("failed to parse float field: %w", err)
	}
	f.data = float32(value)
	return nil
}

func (f *FieldFloat) String() string {
	if f.data == 0 {
		return ""
	}
	return fmt.Sprintf("%05.02f", f.data)
}

func (f *FieldFloat) Value() float32 {
	return f.data
}

func (f *FieldFloat) MarshalJSON() ([]byte, error) {
	if f.data == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(f.Value())
}

func (f *FieldFloat) UnmarshalJSON(b []byte) error {
	var n float32
	if string(b) == "null" {
		f.data = 0
		return nil
	}
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	f.data = n
	return nil
}

/* ------------------------------------------------------------------------------------------------------------------ */

type FieldDate struct {
	data time.Time
}

func (f *FieldDate) parse(data string) error {
	const layout = "2006/01/02" // Customize the layout based on your input format
	data = strings.TrimSpace(data)
	if len(data) == 0 || data == "**NONE**" {
		f.data = time.Time{}
		return nil
	}
	t, err := time.Parse(layout, data)
	if err != nil {
		return fmt.Errorf("failed to parse date field: %w", err)
	}
	f.data = t
	return nil
}

func (f *FieldDate) String() string {
	if f.data.IsZero() {
		return ""
	}
	return f.data.Format("2006/01/02")
}

func (f *FieldDate) Value() time.Time {
	return f.data
}

func (f *FieldDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.String())
}

func (f *FieldDate) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return f.parse(s)
}

/* ------------------------------------------------------------------------------------------------------------------ */

type FieldTime struct {
	data time.Time
}

func (f *FieldTime) parse(data string) error {
	const layout = "2006/01/02 15:04"
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		f.data = time.Time{}
		return nil
	}
	t, err := time.Parse(layout, data)
	if err != nil {
		return fmt.Errorf("failed to parse time field: %w", err)
	}
	f.data = t
	return nil
}

func (f *FieldTime) String() string {
	if f.data.IsZero() {
		return ""
	}
	return f.data.Format("2006/01/02 15:04")
}

func (f *FieldTime) Value() time.Time {
	return f.data
}

func (f *FieldTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.String())
}

func (f *FieldTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return f.parse(s)
}
