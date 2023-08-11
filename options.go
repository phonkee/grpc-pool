/*
 * MIT License
 * Copyright (c) 2023 Peter Vrba
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in the
 * Software without restriction, including without limitation the rights to use, copy,
 * modify, merge, publish, distribute, sublicense, and/or sell copies of the Software,
 * and to permit persons to whom the Software is furnished to do so, subject to the
 * following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
 * NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
 * HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
 * WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
 * DEALINGS IN THE SOFTWARE.
 */

package grpc_pool

import "time"

// Option is a function that can be passed to New to configure the pool.
type Option func(*options) error

// WithAcquireTimeout sets the timeout for acquiring a connection from the pool before retrying again.
// Warning! This is very sensitive value, so please be careful when changing it. It should be very low value.
// Preferably under 100ms. Please set this value to a higher value only if you know what you are doing!
func WithAcquireTimeout(timeout time.Duration) Option {
	return func(o *options) error {
		if timeout <= 0 {
			return ErrInvalidAcquireTimeout
		}
		o.acquireTimeout = timeout
		return nil
	}
}

// WithCleanupInterval sets the interval for cleaning up idle connections.
func WithCleanupInterval(interval time.Duration) Option {
	return func(o *options) error {
		if interval <= 0 {
			return ErrInvalidCleanupInterval
		}
		o.cleanupInterval = interval
		return nil
	}
}

// WithMaxConcurrency sets the maximum number of concurrent method calls on single connection.
func WithMaxConcurrency(max uint) Option {
	return func(o *options) error {
		if max <= 0 {
			return ErrInvalidMaxConcurrency
		}
		o.maxConcurrency = max
		return nil
	}
}

// WithMaxConnections sets the maximum number of connections. This is optional value.
func WithMaxConnections(max uint) Option {
	return func(o *options) error {
		o.maxConnections = max
		return nil
	}
}

// WithMaxIdleConnections sets the maximum number of idle connections. This is optional value.
func WithMaxIdleConnections(max uint) Option {
	return func(o *options) error {
		o.maxIdleConnections = max
		return nil
	}
}

// WithMaxIdleTime sets the maximum idle time of a connection. It is necessary to set this option.
func WithMaxIdleTime(max time.Duration) Option {
	return func(o *options) error {
		if max <= 0 {
			return ErrInvalidMaxIdleTime
		}
		o.maxIdleTime = max
		return nil
	}
}

// WithMaxLifetime sets the maximum lifetime of a connection. It is necessary to set this option to a value lower than zero.
func WithMaxLifetime(max time.Duration) Option {
	return func(o *options) error {
		if max <= 0 {
			return ErrInvalidMaxLifetime
		}
		o.maxLifetime = max
		return nil
	}
}

// newOptions creates a new options object with default values.
func newOptions() *options {
	return &options{
		acquireTimeout:  DefaultAcquireTimeout,
		maxConcurrency:  DefaultMaxConcurrency,
		maxIdleTime:     DefaultMaxIdleTime,
		maxLifetime:     DefaultMaxLifetime,
		cleanupInterval: DefaultCleanupInterval,
	}
}

// options holds all options for the pool.
type options struct {
	acquireTimeout     time.Duration
	maxConcurrency     uint
	maxConnections     uint
	maxLifetime        time.Duration
	maxIdleTime        time.Duration
	maxIdleConnections uint
	cleanupInterval    time.Duration
	nonBlocking        bool
}

// apply applies all options to the options object and returns error if any passed Option returned error
func (o *options) apply(opts ...Option) error {
	for _, opt := range opts {
		if err := opt(o); err != nil {
			return err
		}
	}
	return nil
}
