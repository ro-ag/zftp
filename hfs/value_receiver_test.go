// SPDX-License-Identifier: Apache-2.0

package hfs

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// The public listing APIs (ListDatasets/ListPds/ListSpool, GetJobStatus) hand
// back value types ([]InfoDataset, []InfoJob, ...). For fmt and encoding/json to
// work on those values, the value types — and the Field* types they embed — must
// satisfy fmt.Stringer and json.Marshaler BY VALUE. A pointer-only receiver makes
// fmt print the raw struct and, for json on a non-addressable value, emit {}.
var (
	_ fmt.Stringer = FieldString{}
	_ fmt.Stringer = FieldInt{}
	_ fmt.Stringer = FieldFloat{}
	_ fmt.Stringer = FieldDate{}
	_ fmt.Stringer = FieldTime{}
	_ fmt.Stringer = InfoDataset{}
	_ fmt.Stringer = InfoPdsMember{}
	_ fmt.Stringer = InfoJob{}

	_ json.Marshaler = FieldString{}
	_ json.Marshaler = FieldInt{}
	_ json.Marshaler = FieldFloat{}
	_ json.Marshaler = FieldDate{}
	_ json.Marshaler = FieldTime{}
)

func TestInfoJobValueMarshalsFields(t *testing.T) {
	j := InfoJob{
		Name:   FieldString{data: "MYJOB"},
		JobId:  FieldString{data: "JOB01234"},
		Owner:  FieldString{data: "IBMUSER"},
		Status: FieldString{data: "OUTPUT"},
		Class:  FieldString{data: "A"},
	}
	b, err := json.Marshal(j) // value, exactly as the CLI's emit() passes it
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), `"Name":{}`) {
		t.Fatalf("InfoJob value marshalled Name as {} (pointer-receiver MarshalJSON): %s", b)
	}
	if !strings.Contains(string(b), "MYJOB") {
		t.Fatalf("expected MYJOB in JSON, got %s", b)
	}
}

func TestInfoJobValueSatisfiesStringer(t *testing.T) {
	j := InfoJob{Name: FieldString{data: "MYJOB"}}
	if got := fmt.Sprintf("%v", j); !strings.Contains(got, "MYJOB") {
		t.Fatalf("fmt of InfoJob value did not call String(): %q", got)
	}
}
