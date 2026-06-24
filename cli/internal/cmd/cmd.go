// SPDX-License-Identifier: Apache-2.0

// Package cmd implements the zftp command-line client. Every command is built by
// a newXCmd(deps) constructor and depends only on the injected deps, so commands
// are unit-testable against a fake client with no network, stdout, or terminal.
package cmd

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/ro-ag/zftp.v2"
	"gopkg.in/ro-ag/zftp.v2/hfs"
)

// client is the narrow slice of *FTPSession the commands use. *zftp.FTPSession
// satisfies it structurally; tests inject a fake.
type client interface {
	ListDatasets(expression string) ([]hfs.InfoDataset, error)
	List(expression string) ([]string, error)
	ListPds(expression string) ([]hfs.InfoPdsMember, error)
	ListSpool(expression string) ([]hfs.InfoJob, error)
	Get(remote, local string, mode zftp.TransferType) error
	GetAt(remote, local string, mode zftp.TransferType, offset int64) error
	GetAndGzip(remote, local string, mode zftp.TransferType) error
	Put(local, remote string, mode zftp.TransferType, a ...zftp.DataSpec) error
	PutAt(local, remote string, mode zftp.TransferType, offset int64, a ...zftp.DataSpec) error
	Delete(name string) error
	Mkdir(path string) error
	Rename(from, to string) error
	Chmod(mode, path string) error
	SubmitJCLFile(jclFile string, options ...zftp.JesSpec) (*zftp.JesJob, error)
	GetJobStatus(jobID string) (*hfs.InfoJobDetail, error)
	PurgeJob(jobID string) error
	StatusOf() *zftp.ServerStatus
	System() (string, error)
	Close() error
}

// connOpts carries the resolved connection parameters for the connect factory.
type connOpts struct {
	host, port       string
	user, pass       string
	tls, tlsNoVerify bool
	timeout          time.Duration
	verbosity        int
}

// deps holds every external effect the commands touch — the only seam to the
// outside world. Production wires real implementations; tests wire fakes.
type deps struct {
	connect func(o connOpts) (client, error)
	getenv  func(string) string
	prompt  func() (string, error) // no-echo password reader
	out     io.Writer
	errOut  io.Writer
}

// BuildInfo is the version metadata injected by GoReleaser ldflags via main.
type BuildInfo struct{ Version, Commit, Date string }

// globalFlags are the persistent connection flags shared by all commands.
type globalFlags struct {
	host, port, user   string
	tlsOn, tlsNoVerify bool
	jsonOut            bool
	verbose            int
	timeout            time.Duration
}

// resolvePassword returns ZFTP_PASSWORD if set, else the interactive prompt. It
// never accepts a password flag.
func resolvePassword(getenv func(string) string, prompt func() (string, error)) (string, error) {
	if p := getenv("ZFTP_PASSWORD"); p != "" {
		return p, nil
	}
	return prompt()
}

// emit writes v as indented JSON when jsonOut, else renders the human table.
func emit(d deps, jsonOut bool, v any, table func(w io.Writer)) error {
	if jsonOut {
		enc := json.NewEncoder(d.out)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
	table(d.out)
	return nil
}

// dial resolves the password and opens an authenticated session via d.connect.
func dial(d deps, g *globalFlags) (client, error) {
	pass, err := resolvePassword(d.getenv, d.prompt)
	if err != nil {
		return nil, fmt.Errorf("password: %w", err)
	}
	return d.connect(connOpts{
		host: g.host, port: g.port, user: g.user, pass: pass,
		tls: g.tlsOn, tlsNoVerify: g.tlsNoVerify, timeout: g.timeout, verbosity: g.verbose,
	})
}

// realConnect dials, optionally upgrades to TLS, and logs in.
func realConnect(o connOpts) (client, error) {
	opts := []zftp.Option{zftp.WithSignalHandler()}
	if o.timeout > 0 {
		opts = append(opts, zftp.WithTimeout(o.timeout))
	}
	if o.verbosity > 0 {
		opts = append(opts, zftp.WithLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))))
	}
	s, err := zftp.Open(net.JoinHostPort(o.host, o.port), opts...)
	if err != nil {
		return nil, err
	}
	if o.tls {
		cfg := &tls.Config{ServerName: o.host, InsecureSkipVerify: o.tlsNoVerify} //nolint:gosec // opt-in via --tls-skip-verify
		if err := s.AuthTLS(cfg); err != nil {
			_ = s.Close()
			return nil, err
		}
	}
	if err := s.Login(o.user, o.pass); err != nil {
		_ = s.Close()
		return nil, err
	}
	if o.verbosity == 1 {
		s.SetVerbose(zftp.LogCommand | zftp.LogServer)
	} else if o.verbosity > 1 {
		s.SetVerbose(zftp.LogAll)
	}
	return s, nil
}

// termPrompt reads a password from the controlling terminal with echo disabled.
func termPrompt() (string, error) {
	fmt.Fprint(os.Stderr, "Password: ")
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return string(b), err
}

// newRootCmd assembles the command tree with persistent connection flags.
func newRootCmd(d deps, bi BuildInfo) *cobra.Command {
	g := &globalFlags{}
	root := &cobra.Command{
		Use:           "zftp",
		Short:         "Pure-Go z/OS FTP client",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	pf := root.PersistentFlags()
	pf.StringVarP(&g.host, "host", "H", envOr(d, "ZFTP_HOST", ""), "z/OS FTP host")
	pf.StringVar(&g.port, "port", "21", "control port")
	pf.StringVarP(&g.user, "user", "u", envOr(d, "ZFTP_USER", ""), "login user")
	pf.BoolVar(&g.tlsOn, "tls", false, "AUTH TLS on the control connection")
	pf.BoolVar(&g.tlsNoVerify, "tls-skip-verify", false, "skip TLS certificate verification")
	pf.BoolVar(&g.jsonOut, "json", false, "machine-readable JSON output")
	pf.CountVarP(&g.verbose, "verbose", "v", "increase protocol logging (-v, -vv)")
	pf.DurationVar(&g.timeout, "timeout", 0, "control connection timeout (e.g. 30s)")

	root.AddCommand(
		newVersionCmd(d, bi),
		newLsCmd(d, g),
		newGetCmd(d, g),
		newPutCmd(d, g),
		newRmCmd(d, g),
		newMkdirCmd(d, g),
		newMvCmd(d, g),
		newChmodCmd(d, g),
	)
	root.SetOut(d.out)
	root.SetErr(d.errOut)
	return root
}

func envOr(d deps, key, def string) string {
	if v := d.getenv(key); v != "" {
		return v
	}
	return def
}

// Execute builds production deps and runs the CLI. Returns an error; main maps it
// to a non-zero exit. This is the ONLY place that reads the real environment.
func Execute(bi BuildInfo) error {
	d := deps{
		connect: realConnect,
		getenv:  os.Getenv,
		prompt:  termPrompt,
		out:     os.Stdout,
		errOut:  os.Stderr,
	}
	return newRootCmd(d, bi).Execute()
}
