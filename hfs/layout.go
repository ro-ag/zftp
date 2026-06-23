// SPDX-License-Identifier: Apache-2.0

package hfs

// field describes one fixed-width column in a z/OS listing record. Offsets are
// byte positions into the record as the FTP server renders it. A width of 0
// means "from start to the end of the record" (used for trailing free-form
// columns such as the dataset name).
//
// Declaring the layout as data keeps the column geometry in one auditable place
// instead of scattered slice arithmetic, mirroring the schema-driven approach of
// the ro-ag/parser engine.
type field struct {
	name  string
	start int
	width int
}

// slice extracts the raw (untrimmed) substring for the field. It tolerates
// records shorter than the field by returning whatever is available (or the
// empty string), so a truncated trailing column never panics; callers trim and
// type-convert via the Field* parsers.
func (f field) slice(record string) string {
	if f.start >= len(record) {
		return ""
	}
	if f.width <= 0 {
		return record[f.start:]
	}
	end := min(f.start+f.width, len(record))
	return record[f.start:end]
}
