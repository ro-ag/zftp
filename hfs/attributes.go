// Package hfs provides tools for interacting with the Hierarchical File System (HFS) on z/OS systems.
// It includes functionalities to manage HFS attributes, handle different types of datasets such as partitioned and sequential datasets,
// and interact with the Job Entry Subsystem (JES) spool. The package provides structured data types to represent jobs and job details,
// and includes functions to parse job records and details from the JES spool.

package hfs

import (
	"encoding/json"
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

type FieldInt struct {
	data uint16
}

func (f *FieldInt) parse(data string) error {
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		f.data = 0
		return nil
	}
	value, err := strconv.Atoi(data)
	if err != nil {
		return fmt.Errorf("failed to parse integer field: %v", err)
	}
	f.data = uint16(value)
	return nil
}

func (f *FieldInt) String() string {
	if f.data == 0 {
		return ""
	}
	return strconv.Itoa(int(f.data))
}

func (f *FieldInt) Value() uint16 {
	return f.data
}

func (f *FieldInt) MarshalJSON() ([]byte, error) {
	if f.data == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(f.Value())
}

func (f *FieldInt) UnmarshalJSON(b []byte) error {
	var i int
	if string(b) == "null" {
		f.data = 0
		return nil
	}
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}
	f.data = uint16(i)
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
		return fmt.Errorf("failed to parse float field: %v", err)
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
	if len(data) == 0 {
		f.data = time.Time{}
		return nil
	}
	t, err := time.Parse(layout, data)
	if err != nil {
		return fmt.Errorf("failed to parse date field: %v", err)
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
		return fmt.Errorf("failed to parse time field: %v", err)
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
