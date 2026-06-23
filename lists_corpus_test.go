// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"os"
	"strings"
	"testing"
)

// TestListDatasets_RealWorldCorpus drives the full ListDatasets path (LIST over
// the mock data connection, header skip, per-row parse) over the sanitized
// real-world corpus under ./tests. A single unparseable row would abort the whole
// listing, so this proves the special-state rows (archived, Pseudo Directory,
// "Error determining attributes", …) flow through end to end.
func TestListDatasets_RealWorldCorpus(t *testing.T) {
	body, err := os.ReadFile("tests/dataset_listings/02_special_states.txt")
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	listing := strings.ReplaceAll(string(body), "\n", "\r\n")

	// Count the data rows the server will send (every non-blank line after the
	// column header) so we can assert none were dropped.
	rows := 0
	for i, l := range strings.Split(strings.TrimRight(string(body), "\n"), "\n") {
		if i == 0 || strings.TrimSpace(l) == "" {
			continue
		}
		rows++
	}

	s, srv := dialMock(t)
	srv.DataFor("LIST", "", listing)

	ds, err := s.ListDatasets("HLQ.PROJ.*")
	if err != nil {
		t.Fatalf("ListDatasets aborted on a real-world listing: %v", err)
	}
	if len(ds) != rows {
		t.Fatalf("parsed %d datasets, want %d (a row was dropped)", len(ds), rows)
	}

	// Spot-check classification survived the full pipeline.
	var migrated, pseudo, archived, active int
	for i := range ds {
		switch {
		case ds[i].IsMigrated():
			migrated++
		case ds[i].IsPseudoDirectory():
			pseudo++
		case ds[i].IsArchived():
			archived++
		}
		if ds[i].Active() {
			active++
		}
	}
	if migrated != 2 {
		t.Errorf("migrated rows = %d, want 2", migrated)
	}
	if pseudo != 2 {
		t.Errorf("pseudo-directory rows = %d, want 2", pseudo)
	}
	if archived < 1 {
		t.Errorf("archived rows = %d, want >=1", archived)
	}
	if active == 0 {
		t.Errorf("expected some active datasets in the listing")
	}
}
