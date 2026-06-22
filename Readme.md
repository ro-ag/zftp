# zftp

[![Go Reference](https://pkg.go.dev/badge/gopkg.in/ro-ag/zftp.v2.svg)](https://pkg.go.dev/gopkg.in/ro-ag/zftp.v2)
[![CI](https://github.com/ro-ag/zftp/actions/workflows/ci.yml/badge.svg)](https://github.com/ro-ag/zftp/actions/workflows/ci.yml)
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
  and `WithSignalHandler`.
- `(*FTPSession) Login(user, pass string) error`
- `(*FTPSession) Get(remote, local string, mode TransferType) error` /
  `Put(local, remote string, mode TransferType, a ...DataSpec) error`
- `(*FTPSession) RetrieveIO(remote string, w io.Writer, mode TransferType)` /
  `StoreIO(remote string, r io.Reader, mode TransferType)` — stream without
  touching the local filesystem.
- `(*FTPSession) ListDatasets(pattern string) ([]hfs.InfoDataset, error)`
- `(*FTPSession) ListPds(pattern string) ([]hfs.InfoPdsMember, error)`
- `(*FTPSession) ListSpool(pattern string) ([]hfs.InfoJob, error)`
- `(*FTPSession) SubmitJCL(jcl string, opts ...JesSpec) (*JesJob, error)`
- `(*FTPSession) GetAndGzip(remote, local string, mode TransferType) error` —
  retrieve and gzip in one step.

Full reference: [pkg.go.dev/gopkg.in/ro-ag/zftp.v2](https://pkg.go.dev/gopkg.in/ro-ag/zftp.v2).

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
keeps the library itself dependency-free. It is currently a skeleton; the full
client and downloadable binaries are planned.

## Contributing

Issues and pull requests are welcome. Please open an issue to discuss substantial
changes first.

## License

Licensed under the Apache License 2.0. See [LICENSE](./LICENSE) and
[NOTICE](./NOTICE).
