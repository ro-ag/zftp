# zftp CLI

A command-line client built on the [`gopkg.in/ro-ag/zftp.v2`](../) z/OS FTP
library.

## Why a separate module?

This directory is its own Go module (`github.com/ro-ag/zftp/cli`) with its own
`go.mod`. That keeps the **library** dependency-free: anything the CLI needs for
flag parsing, output formatting, or packaging lives here and never leaks into the
import graph of applications that embed the library.

During development the library is resolved from the parent directory via a
`replace` directive; once `v2.0.0` is tagged that can be dropped.

## Status

This is a **skeleton**. It imports the library and reports its version to prove
the module wiring and dependency isolation. The full client — subcommands
(`ls`, `get`, `put`, `submit`, …) and downloadable, signed binaries produced by
GoReleaser on tag — is tracked as a follow-up and is intentionally out of scope
for the v2 library modernization.

## Build

```sh
cd cli
CGO_ENABLED=0 go build -o zftp .
./zftp -version
```
