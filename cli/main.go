// SPDX-License-Identifier: Apache-2.0

// Command zftp is the command-line client for the zftp z/OS FTP library.
package main

import (
	"fmt"
	"os"

	"github.com/ro-ag/zftp/cli/internal/cmd"
)

// Injected by GoReleaser ldflags: -X main.version / main.commit / main.date.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := cmd.Execute(cmd.BuildInfo{Version: version, Commit: commit, Date: date}); err != nil {
		fmt.Fprintln(os.Stderr, "zftp:", err)
		os.Exit(1)
	}
}
