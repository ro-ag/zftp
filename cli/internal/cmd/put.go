// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"errors"
	"path"

	"github.com/spf13/cobra"
)

// newPutCmd returns the "put" sub-command (STOR). It uploads a local file to a
// remote dataset or path, with optional byte-offset resume.
func newPutCmd(d deps, g *globalFlags) *cobra.Command {
	var ascii bool
	var offset int64
	c := &cobra.Command{
		Use:   "put <local> [remote]",
		Short: "Upload a file or dataset (STOR)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			local := args[0]
			remote := path.Base(local)
			if len(args) == 2 {
				remote = args[1]
			}
			if ascii && offset > 0 {
				return errors.New("--offset is binary-only; ASCII resume is unsupported")
			}
			conn, err := dial(d, g)
			if err != nil {
				return err
			}
			defer conn.Close()
			mode := transferType(ascii)
			if offset > 0 {
				return conn.PutAt(local, remote, mode, offset)
			}
			return conn.Put(local, remote, mode)
		},
	}
	c.Flags().BoolVar(&ascii, "ascii", false, "ASCII (text) transfer; default is binary")
	c.Flags().Int64Var(&offset, "offset", 0, "resume at byte offset (binary only)")
	return c
}
