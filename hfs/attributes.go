package hfs

import (
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
