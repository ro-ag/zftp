// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// newStatCmd returns the stat subcommand that reports system and server status.
func newStatCmd(d deps, g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "stat",
		Short: "Show z/OS system info and server status snapshot",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return withClient(d, g, func(c client) error {
				sys, err := c.System()
				if err != nil {
					return err
				}
				info := map[string]any{"system": sys}
				// StatusOf may return nil (fake or unsupported) — guard every call.
				if st := c.StatusOf(); st != nil {
					if ft, e := st.FileType(); e == nil {
						info["fileType"] = ft
					}
				}
				return emit(d, g.jsonOut, info, func(w io.Writer) {
					fmt.Fprintln(w, sys)
				})
			})
		},
	}
}
