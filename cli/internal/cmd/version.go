// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// newVersionCmd returns the "version" subcommand, which prints build metadata.
func newVersionCmd(d deps, bi BuildInfo) *cobra.Command {
	var jsonOut bool
	c := &cobra.Command{
		Use:   "version",
		Short: "Print version, commit, and build date",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return emit(d, jsonOut, bi, func(w io.Writer) {
				fmt.Fprintf(w, "zftp %s (commit %s, built %s)\n", bi.Version, bi.Commit, bi.Date)
			})
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "JSON output")
	return c
}
