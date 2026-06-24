package cmd

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"gopkg.in/ro-ag/zftp.v2"
)

// *FTPSession must satisfy the narrow client interface the commands consume.
var _ client = (*zftp.FTPSession)(nil)

func TestResolvePassword_EnvWins(t *testing.T) {
	env := map[string]string{"ZFTP_PASSWORD": "secret"}
	got, err := resolvePassword(func(k string) string { return env[k] },
		func() (string, error) { return "", errors.New("prompt must not be called") })
	if err != nil || got != "secret" {
		t.Fatalf("resolvePassword = (%q,%v), want (secret,nil)", got, err)
	}
}

func TestResolvePassword_PromptFallback(t *testing.T) {
	got, err := resolvePassword(func(string) string { return "" },
		func() (string, error) { return "typed", nil })
	if err != nil || got != "typed" {
		t.Fatalf("resolvePassword = (%q,%v), want (typed,nil)", got, err)
	}
}

func TestEmit_JSON(t *testing.T) {
	var buf bytes.Buffer
	d := deps{out: &buf}
	if err := emit(d, true, map[string]string{"a": "b"}, func(w io.Writer) {}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"a": "b"`)) {
		t.Fatalf("json output = %s", buf.String())
	}
}
