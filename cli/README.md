# zftp CLI

A command-line client for z/OS FTP built on the
[`gopkg.in/ro-ag/zftp.v2`](../) library. Every command speaks directly to
z/OS FTP — no middleware, no daemon.

## Install

```sh
go install github.com/ro-ag/zftp/cli@latest
```

> **Submodule / replace caveat:** this directory is its own Go module
> (`github.com/ro-ag/zftp/cli`). During development the library is resolved
> from the parent directory via a `replace` directive in `go.mod`; once
> `v2.0.0` is tagged on `gopkg.in/ro-ag/zftp.v2` that directive is dropped
> and `go install` works without a local clone.

Homebrew tap and `docker run ghcr.io/ro-ag/zftp` are coming with the binary
release pipeline (PR C).

## Environment

| Variable        | Purpose                                      |
|-----------------|----------------------------------------------|
| `ZFTP_HOST`     | z/OS FTP hostname (used as `--host` default) |
| `ZFTP_USER`     | Login user (used as `--user` default)        |
| `ZFTP_PASSWORD` | Login password                               |

`ZFTP_PASSWORD` is read from the environment when set, otherwise the CLI
prompts interactively with echo disabled. A password flag is intentionally
absent — flags appear in shell history; env vars and prompts do not.

## Global flags

| Flag               | Description                                         |
|--------------------|-----------------------------------------------------|
| `-H`, `--host`     | z/OS FTP host (overrides `ZFTP_HOST`)               |
| `-u`, `--user`     | Login user (overrides `ZFTP_USER`)                  |
| `--port`           | Control port (default `21`)                         |
| `--tls`            | AUTH TLS on the control connection                  |
| `--tls-skip-verify`| Skip TLS certificate verification                   |
| `--json`           | Machine-readable JSON output (applies to all cmds)  |
| `-v`, `-vv`        | Protocol logging (`-v` commands/replies, `-vv` all) |
| `--timeout`        | Control connection timeout (e.g. `30s`)             |

## Commands

### `version` — print build metadata

```sh
zftp version
zftp version --json
```

### `ls` — list datasets, PDS members, or HFS entries

```sh
# List datasets matching a pattern (MVS LISTDS)
zftp ls 'USER.*'

# List PDS members
zftp ls 'USER.JCLLIB' --pds

# Raw HFS directory listing
zftp ls '/u/me' --hfs
```

| Flag    | Description                     |
|---------|---------------------------------|
| `--pds` | List PDS members instead        |
| `--hfs` | Raw HFS/LIST output             |

### `get` — download a dataset or file (RETR)

```sh
zftp get 'USER.DATA.FB80' local.dat
zftp get 'USER.DATA.FB80' --ascii local.txt
zftp get 'USER.LARGE' --gzip  large.gz
zftp get 'USER.LARGE' --offset 1048576 resume.dat
```

| Flag       | Description                                |
|------------|--------------------------------------------|
| `--ascii`  | ASCII (text) transfer; default is binary   |
| `--gzip`   | Compress the downloaded stream             |
| `--offset` | Resume at byte offset (binary only)        |

### `put` — upload a file or dataset (STOR)

```sh
zftp put local.dat 'USER.DATA.FB80'
zftp put local.txt 'USER.DATA.FB80' --ascii
zftp put resume.dat 'USER.LARGE' --offset 1048576
```

| Flag       | Description                                |
|------------|--------------------------------------------|
| `--ascii`  | ASCII (text) transfer; default is binary   |
| `--offset` | Resume at byte offset (binary only)        |

### `rm` — delete a dataset or HFS file (DELE)

```sh
zftp rm 'USER.OLD.DATA'
zftp rm '/u/me/tmp.txt'
```

### `mkdir` — create an HFS directory (MKD)

```sh
zftp mkdir '/u/me/newdir'
```

### `mv` — rename a dataset or file (RNFR/RNTO)

```sh
zftp mv 'USER.OLD' 'USER.NEW'
```

### `chmod` — change HFS permissions (SITE CHMOD)

```sh
zftp chmod 750 '/u/me/script.sh'
```

### `stat` — show z/OS system info and server status

```sh
zftp stat
zftp stat --json
```

### `submit` — submit a JCL file to JES

```sh
zftp submit myjob.jcl
```

Returns the job ID (e.g. `JOB12345`).

### `jobs` — list spool jobs

```sh
zftp jobs
zftp jobs 'USER*'
zftp jobs --json
```

### `job` — show a job's status/detail

```sh
zftp job JOB12345
zftp job JOB12345 --json
```

#### `job purge` — purge a job from the spool

```sh
zftp job purge JOB12345
```

## JSON output

Pass `--json` to any command (or use the per-command `--json` flag on
`version`) to receive newline-delimited JSON suitable for `jq`:

```sh
zftp ls 'USER.*' --json | jq '.[].Dsname'
zftp stat --json | jq '.system'
zftp job JOB12345 --json | jq '.JobId'
```

## Follow-ups

`submit --wait` (poll until complete) and `submit --fetch` (download output
spool) are deferred to a later release and not yet implemented.
