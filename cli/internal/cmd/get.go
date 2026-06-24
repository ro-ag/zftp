// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"errors"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/ro-ag/zftp.v2"
)

// transferType maps the --ascii flag to the corresponding zftp transfer mode.
// When ascii is false (the default) binary/image mode is used.
func transferType(ascii bool) zftp.TransferType {
	if ascii {
		return zftp.TypeAscii
	}
	return zftp.TypeImage
}

// newGetCmd returns the "get" sub-command (RETR). It downloads a remote dataset
// or file to a local path, with optional gzip compression or byte-offset resume.
func newGetCmd(d deps, g *globalFlags) *cobra.Command {
	var ascii, gzipOut bool
	var offset int64
	c := &cobra.Command{
		Use:   "get <remote> [local]",
		Short: "Download a dataset or file (RETR)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			remote := args[0]
			local := path.Base(strings.Trim(remote, "'"))
			if len(args) == 2 {
				local = args[1]
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
			switch {
			case gzipOut:
				return conn.GetAndGzip(remote, local, mode)
			case offset > 0:
				return conn.GetAt(remote, local, mode, offset)
			default:
				return conn.Get(remote, local, mode)
			}
		},
	}
	c.Flags().BoolVar(&ascii, "ascii", false, "ASCII (text) transfer; default is binary")
	c.Flags().BoolVar(&gzipOut, "gzip", false, "gzip the downloaded stream")
	c.Flags().Int64Var(&offset, "offset", 0, "resume at byte offset (binary only)")
	return c
}
