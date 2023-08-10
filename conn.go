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

import (
	"google.golang.org/grpc"
	"sync/atomic"
	"time"
)

// newPoolConn creates new pool connection
func newPoolConn(cc *grpc.ClientConn, options *options) *poolConn {
	now := time.Now()

	// now create new pool connection
	result := &poolConn{
		Created:    now,
		Chan:       make(chan *grpc.ClientConn, options.maxConcurrency),
		Conn:       cc,
		LastChange: atomic.Pointer[time.Time]{},
		Usage:      new(Counter),
	}

	result.LastChange.Store(&now)

	// now add all necessary connections
	for i := 0; i < options.maxConcurrency; i++ {
		result.Chan <- cc
	}
	return result
}

// poolConn holds information about single connection in the pool with additional information and also queue of
// connections. This is used to implement the pool.
type poolConn struct {
	// Created is the time when the connection was created
	Created time.Time
	// Conn is the actual connection, it is held here, so we can access it easily
	Conn *grpc.ClientConn
	// Chan is the channel of prepared connections, it is used in main select
	Chan chan *grpc.ClientConn
	// LastChange is the time when the last change happened, it is used to determine if the connection is idle
	LastChange atomic.Pointer[time.Time]
	// Usage is the counter of how many times the connection was used, usable only in stats
	Usage *Counter
}

// isFull means channel has all connections
func (p *poolConn) isFull() bool {
	return len(p.Chan) == cap(p.Chan)
}

// close closes the connection and also closes the channel
// does all the necessary cleanup
func (p *poolConn) close() error {
	// consume whole channel
	close(p.Chan)
	for _ = range p.Chan {
		// no-op, just drain channel
	}
	return p.Conn.Close()
}

// stats returns the stats of the connection, depending on options
func (p *poolConn) stats(opts *options) PoolConnStats {
	// store channel length to be consistent in results (concurrency issues)
	l := len(p.Chan)
	return PoolConnStats{
		Target:     p.Conn.Target(),
		Created:    p.Created,
		Deadline:   p.Created.Add(opts.maxLifetime),
		LastChange: *(p.LastChange.Load()),
		Working:    opts.maxConcurrency - l,
		Idle:       l,
	}
}
