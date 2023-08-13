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

// newConn creates new pool connection
func newConn(cc *grpc.ClientConn, options *options) *conn {
	// now create new pool connection
	result := &conn{
		Created:        time.Now(),
		ClientConnChan: make(chan *grpc.ClientConn, options.maxConcurrency),
		ClientConn:     cc,
		LastChange:     atomic.Pointer[time.Time]{},
		Usage:          new(atomic.Uint64),
	}

	result.LastChange.Store(ptrTo(time.Now()))

	// now add all necessary connections
	for i := 0; uint(i) < options.maxConcurrency; i++ {
		result.ClientConnChan <- cc
	}
	return result
}

// conn holds information about single connection in the pool with additional information
// it also holds buffered channel with pointers to connections. This is used to hold track of borrowed connections
// and makes things easy.
type conn struct {
	// Created is the time when the connection was created
	Created time.Time
	// ClientConn is the actual connection, it is held here, so we can access it easily
	ClientConn *grpc.ClientConn
	// ClientConnChan is the channel of prepared connections, it is used in main select
	// when connection is created, we create channel with maxConcurrency capacity and fill it with connectiion,
	// this is necessary for pool algorithm to work
	ClientConnChan chan *grpc.ClientConn
	// LastChange is the time when the last change happened, it is used to determine if the connection is idle
	LastChange atomic.Pointer[time.Time]
	// Usage is the counter of how many times the connection was used, usable only in stats
	Usage *atomic.Uint64
}

// isFull means channel has all connections
func (p *conn) isFull() bool {
	return len(p.ClientConnChan) == cap(p.ClientConnChan)
}

// close closes the connection and also closes the channel
// does all the necessary cleanup
func (p *conn) close() error {
	// consume whole channel
	close(p.ClientConnChan)
	for range p.ClientConnChan {
		// no-op, just drain channel
	}
	return p.ClientConn.Close()
}

// stats returns the stats of the connection, depending on options
func (p *conn) stats(opts *options) ConnStats {
	// store channel length to be consistent in results (concurrency issues)
	l := uint(len(p.ClientConnChan))
	return ConnStats{
		Target:       p.ClientConn.Target(),
		Created:      p.Created,
		Deadline:     p.Created.Add(opts.maxLifetime),
		LastChange:   *(p.LastChange.Load()),
		WorkingConns: opts.maxConcurrency - l,
		IdleConns:    l,
		Usage:        p.Usage.Load(),
	}
}
