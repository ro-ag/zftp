package zftp

import "time"

// Option configures behavior for Open and passive connections.
type Option func(*dialOptions)

// dialOptions holds configuration for network dialing.
type dialOptions struct {
	DialTimeout     time.Duration
	KeepAlivePeriod time.Duration
}

// apply runs the option functions on a dialOptions struct.
func (o *dialOptions) apply(opts []Option) {
	for _, fn := range opts {
		fn(o)
	}
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
