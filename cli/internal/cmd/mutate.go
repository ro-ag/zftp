// SPDX-License-Identifier: Apache-2.0
package cmd

import "github.com/spf13/cobra"

// withClient runs fn against a freshly dialed session, closing it after.
func withClient(d deps, g *globalFlags, fn func(c client) error) error {
	c, err := dial(d, g)
	if err != nil {
		return err
	}
	defer c.Close()
	return fn(c)
}

// newRmCmd returns the rm subcommand that deletes a dataset or HFS file.
func newRmCmd(d deps, g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use: "rm <path>", Short: "Delete a dataset or file (DELE)", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			return withClient(d, g, func(c client) error { return c.Delete(a[0]) })
		},
	}
}

// newMkdirCmd returns the mkdir subcommand that creates an HFS directory.
func newMkdirCmd(d deps, g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use: "mkdir <path>", Short: "Create a directory (MKD)", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			return withClient(d, g, func(c client) error { return c.Mkdir(a[0]) })
		},
	}
}

// newMvCmd returns the mv subcommand that renames a dataset or HFS file.
func newMvCmd(d deps, g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use: "mv <from> <to>", Short: "Rename a dataset or file (RNFR/RNTO)", Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, a []string) error {
			return withClient(d, g, func(c client) error { return c.Rename(a[0], a[1]) })
		},
	}
}

// newChmodCmd returns the chmod subcommand that changes HFS permissions.
func newChmodCmd(d deps, g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use: "chmod <mode> <path>", Short: "Change HFS permissions (SITE CHMOD)", Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, a []string) error {
			return withClient(d, g, func(c client) error { return c.Chmod(a[0], a[1]) })
		},
	}
}
