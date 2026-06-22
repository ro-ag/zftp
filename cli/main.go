// SPDX-License-Identifier: Apache-2.0

// Command zftp is a (skeleton) command-line client built on the
// gopkg.in/ro-ag/zftp.v2 library. It lives in its own module so the library
// itself stays dependency-free; the full CLI (subcommands + release binaries) is
// tracked separately.
package main

import (
	"flag"
	"fmt"
	"os"

	zftp "gopkg.in/ro-ag/zftp.v2"
)

// version is overwritten at release time via -ldflags "-X main.version=...".
var version = "0.0.0-dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("zftp %s\n", version)
		return
	}

	fmt.Fprintln(os.Stderr, "zftp: command-line client for the z/OS FTP library gopkg.in/ro-ag/zftp.v2 (skeleton)")
	fmt.Fprintf(os.Stderr, "default transfer type: %s\n", zftp.TypeBinary.Name())
	fmt.Fprintln(os.Stderr, "the full CLI is not built yet; run with -version for the build version.")
}
