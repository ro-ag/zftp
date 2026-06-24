// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"bytes"
	"testing"

	"gopkg.in/ro-ag/zftp.v2"
	"gopkg.in/ro-ag/zftp.v2/hfs"
)

// fakeClient is a test double that satisfies the client interface. Fields are
// pre-loaded with canned responses; calls logs each method invocation for
// sequence assertions.
type fakeClient struct {
	calls     []string // method:arg log for sequence assertions
	datasets  []hfs.InfoDataset
	listLines []string
	pds       []hfs.InfoPdsMember
	jobs      []hfs.InfoJob
	jobDetail *hfs.InfoJobDetail
	submitJob *zftp.JesJob
	status    *zftp.ServerStatus
	system    string
	err       error // returned by the next mutating call when set
}

func (f *fakeClient) ListDatasets(e string) ([]hfs.InfoDataset, error) {
	f.calls = append(f.calls, "ListDatasets:"+e)
	return f.datasets, f.err
}
func (f *fakeClient) List(e string) ([]string, error) {
	f.calls = append(f.calls, "List:"+e)
	return f.listLines, f.err
}
func (f *fakeClient) ListPds(e string) ([]hfs.InfoPdsMember, error) {
	f.calls = append(f.calls, "ListPds:"+e)
	return f.pds, f.err
}
func (f *fakeClient) ListSpool(e string) ([]hfs.InfoJob, error) {
	f.calls = append(f.calls, "ListSpool:"+e)
	return f.jobs, f.err
}
func (f *fakeClient) Get(r, l string, m zftp.TransferType) error {
	f.calls = append(f.calls, "Get:"+r+"->"+l)
	return f.err
}
func (f *fakeClient) GetAt(r, l string, m zftp.TransferType, o int64) error {
	f.calls = append(f.calls, "GetAt:"+r)
	return f.err
}
func (f *fakeClient) GetAndGzip(r, l string, m zftp.TransferType) error {
	f.calls = append(f.calls, "GetAndGzip:"+r)
	return f.err
}
func (f *fakeClient) Put(l, r string, m zftp.TransferType, a ...zftp.DataSpec) error {
	f.calls = append(f.calls, "Put:"+l+"->"+r)
	return f.err
}
func (f *fakeClient) PutAt(l, r string, m zftp.TransferType, o int64, a ...zftp.DataSpec) error {
	f.calls = append(f.calls, "PutAt:"+l)
	return f.err
}
func (f *fakeClient) Delete(n string) error {
	f.calls = append(f.calls, "Delete:"+n)
	return f.err
}
func (f *fakeClient) Mkdir(p string) error {
	f.calls = append(f.calls, "Mkdir:"+p)
	return f.err
}
func (f *fakeClient) Rename(a, b string) error {
	f.calls = append(f.calls, "Rename:"+a+"->"+b)
	return f.err
}
func (f *fakeClient) Chmod(m, p string) error {
	f.calls = append(f.calls, "Chmod:"+m+" "+p)
	return f.err
}
func (f *fakeClient) SubmitJCLFile(j string, o ...zftp.JesSpec) (*zftp.JesJob, error) {
	f.calls = append(f.calls, "SubmitJCLFile:"+j)
	return f.submitJob, f.err
}
func (f *fakeClient) GetJobStatus(id string) (*hfs.InfoJobDetail, error) {
	f.calls = append(f.calls, "GetJobStatus:"+id)
	return f.jobDetail, f.err
}
func (f *fakeClient) PurgeJob(id string) error {
	f.calls = append(f.calls, "PurgeJob:"+id)
	return f.err
}
func (f *fakeClient) StatusOf() *zftp.ServerStatus { return f.status }
func (f *fakeClient) System() (string, error)      { return f.system, f.err }
func (f *fakeClient) Close() error {
	f.calls = append(f.calls, "Close")
	return nil
}

// runCLI builds a root command wired to fake (via deps.connect) and the given
// env, executes argv, and returns captured stdout.
func runCLI(t *testing.T, fake *fakeClient, env map[string]string, argv ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	d := deps{
		connect: func(connOpts) (client, error) { return fake, nil },
		getenv:  func(k string) string { return env[k] },
		prompt:  func() (string, error) { return "pw", nil },
		out:     &out, errOut: &out,
	}
	root := newRootCmd(d, BuildInfo{Version: "test"})
	root.SetArgs(argv)
	err := root.Execute()
	return out.String(), err
}
