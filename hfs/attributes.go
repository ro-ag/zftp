package hfs

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type StringField struct {
	data string
}

func (f *StringField) parse(data string) error {
	f.data = strings.TrimSpace(data)
	return nil
}

func (f *StringField) String() string {
	return f.data
}

func (f *StringField) Value() string {
	return f.data
}

type IntField struct {
	data uint16
}

func (f *IntField) parse(data string) error {
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		return nil
	}
	value, err := strconv.Atoi(data)
	if err != nil {
		return fmt.Errorf("failed to parse integer field: %v", err)
	}
	f.data = uint16(value)
	return nil
}

func (f *IntField) String() string {
	if f.data == 0 {
		return ""
	}
	return strconv.Itoa(int(f.data))
}

func (f *IntField) Value() uint16 {
	return f.data
}

type FloatField struct {
	data float32
}

func (f *FloatField) parse(data string) error {
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		return nil
	}
	value, err := strconv.ParseFloat(data, 32)
	if err != nil {
		return fmt.Errorf("failed to parse float field: %v", err)
	}
	f.data = float32(value)
	return nil
}

func (f *FloatField) String() string {
	return fmt.Sprintf("%05.02f", f.data)
}

func (f *FloatField) Value() float32 {
	return f.data
}

type DateField struct {
	data time.Time
}

func (f *DateField) parse(data string) error {
	const layout = "2006/01/02" // Customize the layout based on your input format
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		return nil
	}
	t, err := time.Parse(layout, data)
	if err != nil {
		return fmt.Errorf("failed to parse date field: %v", err)
	}
	f.data = t
	return nil
}

func (f *DateField) String() string {
	if f.data.IsZero() {
		return ""
	}
	return f.data.Format("2006/01/02")
}

func (f *DateField) Value() time.Time {
	return f.data
}

type TimeField struct {
	data time.Time
}

func (f *TimeField) parse(data string) error {
	const layout = "2006/01/02 15:04"
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		return nil
	}
	t, err := time.Parse(layout, data)
	if err != nil {
		return fmt.Errorf("failed to parse time field: %v", err)
	}
	f.data = t
	return nil
}

func (f *TimeField) String() string {
	return f.data.Format("2006/01/02 15:04")
}

func (f *TimeField) Value() time.Time {
	return f.data
}
