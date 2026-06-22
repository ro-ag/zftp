// SPDX-License-Identifier: Apache-2.0

package zftp

import "time"

// Option configures behavior for Open and passive connections.
type Option func(*dialOptions)

// dialOptions holds configuration for network dialing.
type dialOptions struct {
	DialTimeout     time.Duration
	KeepAlivePeriod time.Duration
	dialer          Dialer
	signalHandler   bool
}

// apply runs the option functions on a dialOptions struct.
func (o *dialOptions) apply(opts []Option) {
	for _, fn := range opts {
		fn(o)
	}
}

// WithDialer supplies a custom Dialer for the control connection. When unset,
// a standard *net.Dialer is used with the configured timeout and keep-alive.
// Primarily a testing seam (inject an in-process server), but also useful for
// proxies or custom transports.
func WithDialer(d Dialer) Option {
	return func(o *dialOptions) { o.dialer = d }
}

// WithSignalHandler installs a process-wide SIGINT/SIGTERM handler that closes
// the session and then calls os.Exit.
//
// It is opt-in: a library must not hijack the host application's signal handling
// or terminate its process by default. Enable it only in standalone command-line
// tools that want Ctrl-C to tear the session down cleanly.
func WithSignalHandler() Option {
	return func(o *dialOptions) { o.signalHandler = true }
}

// WithTimeout sets the timeout for establishing connections.
func WithTimeout(d time.Duration) Option {
	return func(o *dialOptions) { o.DialTimeout = d }
}

// WithKeepAlive sets TCP keep-alive with the given period.
// A zero duration disables keep-alives.
func WithKeepAlive(d time.Duration) Option {
	return func(o *dialOptions) { o.KeepAlivePeriod = d }
}
