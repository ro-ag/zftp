// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// newSubmitCmd returns the submit subcommand that sends a JCL file to JES.
func newSubmitCmd(d deps, g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "submit <jclfile>",
		Short: "Submit a JCL file to JES (returns the job id)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			return withClient(d, g, func(c client) error {
				job, err := c.SubmitJCLFile(a[0])
				if err != nil {
					return err
				}
				return emit(d, g.jsonOut, job, func(w io.Writer) { fmt.Fprintln(w, job.ID) })
			})
		},
	}
}

// newJobsCmd returns the jobs subcommand that lists spool jobs.
func newJobsCmd(d deps, g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "jobs [pattern]",
		Short: "List spool jobs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			pat := "*"
			if len(a) == 1 {
				pat = a[0]
			}
			return withClient(d, g, func(c client) error {
				js, err := c.ListSpool(pat)
				if err != nil {
					return err
				}
				return emit(d, g.jsonOut, js, func(w io.Writer) {
					tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
					fmt.Fprintln(tw, "JOBNAME\tJOBID\tOWNER\tSTATUS\tCLASS")
					for i := range js {
						fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
							js[i].Name.String(), js[i].JobId.String(),
							js[i].Owner.String(), js[i].Status.String(),
							js[i].Class.String())
					}
					tw.Flush()
				})
			})
		},
	}
}

// newJobCmd returns the job subcommand that shows a job's status/detail and
// hosts the job purge subcommand.
//
// Fix 1: InfoJob has no .String() on the struct value — call fields explicitly.
// Fix 2: emit jd.Job() (InfoJob, exported fields) not jd (*InfoJobDetail, unexported).
func newJobCmd(d deps, g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job <id>",
		Short: "Show a job's status/detail",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			return withClient(d, g, func(c client) error {
				jd, err := c.GetJobStatus(a[0])
				if err != nil {
					return err
				}
				job := jd.Job()
				// Emit job (InfoJob, exported json-tagged fields) for JSON so the
				// output is a real object rather than {}. For the human render,
				// print the fields explicitly — InfoJob.String() exists only as a
				// pointer receiver (*InfoJob) and is not called on the value here.
				return emit(d, g.jsonOut, job, func(w io.Writer) {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
						job.Name.String(), job.JobId.String(),
						job.Owner.String(), job.Status.String(),
						job.Class.String())
				})
			})
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "purge <id>",
		Short: "Purge a job from the spool (DELE under FILETYPE=JES)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			return withClient(d, g, func(c client) error { return c.PurgeJob(a[0]) })
		},
	})
	return cmd
}
