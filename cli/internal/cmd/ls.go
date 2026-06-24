// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// newLsCmd returns the "ls" subcommand, which lists datasets (default), PDS
// members (--pds), or raw HFS entries (--hfs).
func newLsCmd(d deps, g *globalFlags) *cobra.Command {
	var pds, hfsMode bool
	c := &cobra.Command{
		Use:   "ls [pattern]",
		Short: "List datasets (default), PDS members (--pds), or HFS entries (--hfs)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pattern := "*"
			if len(args) == 1 {
				pattern = args[0]
			}
			c, err := dial(d, g)
			if err != nil {
				return err
			}
			defer c.Close()
			switch {
			case hfsMode:
				lines, err := c.List(pattern)
				if err != nil {
					return err
				}
				return emit(d, g.jsonOut, lines, func(w io.Writer) {
					for _, l := range lines {
						fmt.Fprintln(w, l)
					}
				})
			case pds:
				ms, err := c.ListPds(pattern)
				if err != nil {
					return err
				}
				return emit(d, g.jsonOut, ms, func(w io.Writer) {
					tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
					fmt.Fprintln(tw, "NAME\tVV.MM\tCHANGED\tSIZE\tID")
					for i := range ms {
						fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", ms[i].Name.String(), ms[i].VvMm.String(), ms[i].Changed.String(), ms[i].Size.String(), ms[i].Id.String())
					}
					tw.Flush()
				})
			default:
				ds, err := c.ListDatasets(pattern)
				if err != nil {
					return err
				}
				return emit(d, g.jsonOut, ds, func(w io.Writer) {
					tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
					fmt.Fprintln(tw, "NAME\tVOLUME\tRECFM\tLRECL\tDSORG\tSTATE")
					for i := range ds {
						fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", ds[i].Name(), ds[i].Volume.String(), ds[i].Recfm.String(), ds[i].Lrecl.String(), ds[i].Dsorg.String(), ds[i].State())
					}
					tw.Flush()
				})
			}
		},
	}
	c.Flags().BoolVar(&pds, "pds", false, "list PDS members")
	c.Flags().BoolVar(&hfsMode, "hfs", false, "raw HFS/LIST output")
	return c
}
