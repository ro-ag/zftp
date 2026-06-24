// SPDX-License-Identifier: Apache-2.0

package hfs

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// The public listing APIs (ListDatasets/ListPds/ListSpool, GetJobStatus) hand
// back value types ([]InfoDataset, []InfoJob, ...), so the value types must
// satisfy fmt.Stringer and json.Marshaler BY VALUE. The compile-time guarantees
// live in the source files (attributes.go/dataset.go/partitioned.go/spool.go);
// these tests exercise the runtime behavior — that a value actually marshals its
// fields rather than emitting {}, and prints via String().

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
