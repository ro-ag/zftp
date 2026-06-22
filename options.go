// SPDX-License-Identifier: Apache-2.0

package zftp

import "time"

// Option configures behavior for Open and passive connections.
type Option func(*dialOptions)

// dialOptions holds configuration for network dialing.
type dialOptions struct {
	DialTimeout     time.Duration
	KeepAlivePeriod time.Duration
	ReplyTimeout    time.Duration
	dialer          Dialer
	signalHandler   bool
}

// defaultReplyTimeout bounds the wait for a post-transfer control reply. It is
// generous (a well-behaved z/OS server answers in well under a second) but finite,
// so a lost reply cannot hang the caller forever. It mirrors the z/OS DATATIMEOUT
// default of 120s.
const defaultReplyTimeout = 120 * time.Second

// replyTimeout returns the configured post-transfer reply timeout, falling back to
// defaultReplyTimeout when unset.
func (o dialOptions) replyTimeout() time.Duration {
	if o.ReplyTimeout <= 0 {
		return defaultReplyTimeout
	}
	return o.ReplyTimeout
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

// WithReplyTimeout bounds how long the client waits for the terminal reply (the
// 226/250 that follows a data transfer). z/OS sends that reply asynchronously to
// the data-connection close, and it can be lost — for example a NAT dropping an
// idle control link during a long transfer — so bounding the read keeps a lost
// reply from hanging the caller. Defaults to 120s; a non-positive duration
// restores the default.
func WithReplyTimeout(d time.Duration) Option {
	return func(o *dialOptions) { o.ReplyTimeout = d }
}
