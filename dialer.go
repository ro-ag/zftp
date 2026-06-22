// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"context"
	"net"
)

// Dialer establishes the control connection for an FTP session. The standard
// library's *net.Dialer satisfies this interface, so it is used by default.
//
// It is the one narrow input seam the session is built around: tests (and
// advanced callers) supply their own implementation via WithDialer to exercise
// the client against an in-process server with no real network.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}
