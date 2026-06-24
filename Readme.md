# zftp

[![Go Reference](https://pkg.go.dev/badge/gopkg.in/ro-ag/zftp.v2.svg)](https://pkg.go.dev/gopkg.in/ro-ag/zftp.v2)
[![CI](https://github.com/ro-ag/zftp/actions/workflows/ci.yml/badge.svg)](https://github.com/ro-ag/zftp/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/ro-ag/zftp)](https://github.com/ro-ag/zftp/releases/latest)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](./LICENSE)

A **pure-Go FTP client specialized for IBM z/OS mainframes** — datasets, JES
spool, HFS attributes, SITE parameters, and mainframe transfer modes. No cgo, no
external dependencies.

```bash
go get gopkg.in/ro-ag/zftp.v2
```

> v2 is the current line. It is a clean, SemVer-correct break from v1: the public
> API returns concrete types instead of interfaces. Existing v1 tags are
> unaffected.

## Install (CLI binary)

The `zftp` command-line client is distributed as a prebuilt binary via GitHub
Releases, Homebrew, and Docker. A tag-triggered release workflow builds
multi-platform archives, a multi-arch Docker image on GHCR, and a Homebrew cask.

### GitHub Releases

Download the archive for your OS/arch from the
[Releases page](https://github.com/ro-ag/zftp/releases/latest) and extract
the `zftp` binary:

```bash
# archives are named zftp_<version>_<os>_<arch>.tar.gz
# os: linux|darwin|windows   arch: amd64|arm64   (.zip on windows)
VERSION=2.0.0
curl -L https://github.com/ro-ag/zftp/releases/download/v${VERSION}/zftp_${VERSION}_linux_amd64.tar.gz \
  | tar -xz zftp
sudo mv zftp /usr/local/bin/
```

### Homebrew

```bash
brew tap ro-ag/tap        # only needed the first time
brew install --cask ro-ag/tap/zftp
```

The tap publishes a **cask** that installs the prebuilt binary — no build from
source.

### Docker

```bash
docker run --rm ghcr.io/ro-ag/zftp:latest version
```

The image is multi-arch (amd64/arm64) and hosted on GHCR.

### go install (development builds only)

```bash
go install github.com/ro-ag/zftp/cli@latest
```

> **Note:** `cli/` uses a `replace` directive pointing to `../` for the
> library, so `go install` from a clean module checkout is not wired yet.
> For production use, prefer the release binaries, Homebrew, or Docker above.

## Features

- **Session management** — connect, authenticate, and run z/OS FTP commands; a
  custom `ReturnError` carries the received and expected reply codes.
- **File transfer** — ASCII and binary (image) modes, end-of-line conversion for
  ASCII transfers, and offset/`REST`-based resume.
- **Datasets** — list with full attributes (volume, unit, RECFM, LRECL, BLKSIZE,
  DSORG) and classify sequential, partitioned (PDS), migrated, not-mounted, and
  VSAM datasets.
- **JES** — submit jobs (JCL) and parse the spool, including interface levels 1
  and 2, return codes, ABENDs, and JCL errors.
- **SITE / status** — read server status via `XSTA` (`StatusOf`) and set dataset
  allocation attributes via `SITE` (`SetStatusOf`, `SetDataSpecs`).
- **Passive mode** and TLS (`AUTH TLS`).

## Quick start

```go
package main

import (
	"fmt"
	"log"

	zftp "gopkg.in/ro-ag/zftp.v2"
)

func main() {
	// Address is host:port.
	s, err := zftp.Open("mainframe.example.com:21")
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	if err := s.Login("USER", "PASSWORD"); err != nil {
		log.Fatal(err)
	}

	// List datasets with their attributes.
	datasets, err := s.ListDatasets("USER.*")
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range datasets {
		fmt.Printf("%-44s recfm=%-4s lrecl=%s\n", d.Name(), d.Recfm.String(), d.Lrecl.String())
	}

	// Download a member in binary mode.
	if err := s.Get("USER.SOURCE(MEMBER)", "member.txt", zftp.TypeBinary); err != nil {
		log.Fatal(err)
	}
}
```

A compile-checked version of this snippet lives in
[`example_test.go`](./example_test.go).

## Commonly used API

- `Open(address string, opts ...Option) (*FTPSession, error)` — open a session
  to `host:port`. Options include `WithTimeout`, `WithKeepAlive`, `WithDialer`,
  `WithSignalHandler`, and `WithLogger` (see [Logging](#logging)).
- `(*FTPSession) Login(user, pass string) error`
- `(*FTPSession) Get(remote, local string, mode TransferType) error` /
  `Put(local, remote string, mode TransferType, a ...DataSpec) error`
- `(*FTPSession) RetrieveIO(remote string, w io.Writer, mode TransferType) (int64, error)` /
  `StoreIO(remote string, r io.Reader, mode TransferType) (int64, error)` — stream
  without touching the local filesystem.
- `(*FTPSession) ListDatasets(pattern string) ([]hfs.InfoDataset, error)`
- `(*FTPSession) ListPds(pattern string) ([]hfs.InfoPdsMember, error)`
- `(*FTPSession) ListSpool(pattern string) ([]hfs.InfoJob, error)`
- `(*FTPSession) SubmitJCL(jcl string, opts ...JesSpec) (*JesJob, error)`
- `(*FTPSession) GetAndGzip(remote, local string, mode TransferType) error` —
  retrieve and gzip in one step.

Full reference: [pkg.go.dev/gopkg.in/ro-ag/zftp.v2](https://pkg.go.dev/gopkg.in/ro-ag/zftp.v2).

## Logging

zftp logs through the standard library's [`log/slog`](https://pkg.go.dev/log/slog).
Logging is **per session** and **silent by default** — nothing is emitted until you
opt in with `SetVerbose`.

`SetVerbose` selects which trace categories a session emits: a bitmask of
`LogServer`, `LogPassive`, `LogCommand`, `LogDebug` (or `LogAll`; `NoLog` disables
them). Route the records into your own logger with `WithLogger` at open time, or
`SetLogger` at runtime:

```go
h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
s, err := zftp.Open("host:21", zftp.WithLogger(slog.New(h)))
// ...
s.SetVerbose(zftp.LogCommand | zftp.LogServer)
```

With no `WithLogger`, zftp falls back to `slog.Default()`.

**Record shape.** Every record carries `component=zftp`. The four trace categories
are emitted at `slog.LevelDebug` with a `category` attribute
(`server`/`passive`/`command`/`debug`); warnings at `slog.LevelWarn` and errors at
`slog.LevelError`. Your handler's own level still applies on top of `SetVerbose`, so
a handler set to `LevelInfo` drops the trace lines while keeping warnings and errors.

**Bringing your own logger (zap, zerolog).** The core depends only on `log/slog`, so
any logger that exposes a `slog.Handler` plugs in — the bridge package lives in your
application's `go.mod`, and zftp stays dependency-free:

```go
// zap — via a zap→slog handler (go.uber.org/zap/exp/zapslog)
s, _ := zftp.Open(addr, zftp.WithLogger(slog.New(zapHandler)))     // zapHandler is a slog.Handler

// zerolog — via a zerolog→slog handler (e.g. samber/slog-zerolog)
s, _ := zftp.Open(addr, zftp.WithLogger(slog.New(zerologHandler))) // zerologHandler is a slog.Handler
```

Check each bridge's current release for its exact handler constructor.

## Testing without a mainframe

The package ships an in-process mock z/OS FTP server (`internal/mockzos`) used by
the unit tests, so the full client — dial, passive negotiation, data transfer,
multiline reply parsing, dataset/JES parsing — is exercised over loopback with no
real host. The fixed-width parsers are verified by exact-match golden tests
against captured z/OS output (`hfs/testdata`). Run them with:

```bash
go test -race ./...
```

Integration tests that require a live host are skipped unless `ZFTP_HOSTNAME`,
`ZFTP_USERNAME`, and `ZFTP_PASSWORD` are set.

## Command-line client

A CLI built on this library lives in [`cli/`](./cli) as a separate module, which
keeps the library itself dependency-free. See the [Install](#install-cli-binary)
section above for prebuilt binaries, Homebrew, and Docker.

## Contributing

Issues and pull requests are welcome. Please open an issue to discuss substantial
changes first.

## License

Licensed under the Apache License 2.0. See [LICENSE](./LICENSE) and
[NOTICE](./NOTICE).
