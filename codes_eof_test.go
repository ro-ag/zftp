// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"

	"gopkg.in/ro-ag/zftp.v2/internal/log"
)

// M4: a multiline reply whose stream ends after the opening line but before the
// terminator line is truncated — the control connection died mid-reply. check
// must surface io.ErrUnexpectedEOF (so the caller closes the desynchronized
// session) rather than returning the partial reply as a successful (nil) result,
// even when the opening code equals the expected code.
func TestCheck_EOFAfterOpeningLineIsError(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("220-line one\r\n220-line two\r\n"))
	_, err := CodeSvcReadySoon.check(r, log.New(nil, log.None))
	if err == nil {
		t.Fatal("EOF after opening line returned nil: truncated reply reported as success")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("want io.ErrUnexpectedEOF, got %v", err)
	}
}

// A complete single-line reply terminated normally must still succeed (guard that
// the fix does not over-trigger on a clean terminator followed by EOF).
func TestCheck_CompleteReplyStillSucceeds(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("220 service ready\r\n"))
	msg, err := CodeSvcReadySoon.check(r, log.New(nil, log.None))
	if err != nil {
		t.Fatalf("complete reply returned error: %v", err)
	}
	if !strings.Contains(msg, "service ready") {
		t.Fatalf("unexpected message: %q", msg)
	}
}
