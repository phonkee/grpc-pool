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
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// These chosen constants are in strict order, so we can use them to check which channel was selected
// Warning: do not change the order of these constants, it will break main loop select.
const (
	// ChosenContextDeadline is returned from Select when context deadline is reached
	ChosenContextDeadline = 0
	// ChosenAcquireTimeout is returned from Select when acquire timeout is reached
	ChosenAcquireTimeout = 1
)

// New creates a new pool of gRPC connections.
//
// DialFUnc is required because pool cannot work without it.
// Options can be passed to configure the pool.
func New(dialFunc DialFunc, opts ...Option) (*Pool, error) {
	// first instantiate default options with dialFunc
	o, err := newOptions(dialFunc)
	if err != nil {
		return nil, err
	}
	// now apply all passed Options
	if err := o.apply(opts...); err != nil {
		return nil, err
	}
	result := &Pool{
		options: o,
		storage: make(map[*grpc.ClientConn]*conn),
		mutex:   &sync.RWMutex{},
	}

	// run background cleanup goroutine
	// this goroutine will clean up all the connections that should be cleaned
	// this task is mandatory, however you can change the interval when it's running
	// I still suggest to keep the interval low
	go func() {
		// create ticker with cleanup interval
		tick := time.NewTicker(result.options.cleanupInterval)
		for {
			select {
			// if pool is closed, stop the goroutine
			case <-result.close:
				return
			// if ticker ticks, run cleanup
			case <-tick.C:
				result.cleanupConnections()
			}
		}
	}()

	return result, nil
}

// Pool implementation
type Pool struct {
	// storage stores all the connections (even to be closed etc...)
	storage map[*grpc.ClientConn]*conn
	// slice of connections that are currently available in correct order (used because map is not ordered)
	// this slice is used to select on.
	conns []*conn
	// mutex to protect read/write access to storage and conns
	mutex *sync.RWMutex
	// pool options
	options *options
	// close channel is used to close the pool
	close chan struct{}
	// isClosed is used to check if pool is isClosed
	isClosed atomic.Bool
}

// Acquire acquires single connection from the pool.
// It checks if there is any connection available, and if not, it will dial new connection.
//
// It uses unsafe, but uses it in a very safe way.
// It is safe to use it in concurrent environment.
//
// Do not forget to Release the connection when you are done with it. Otherwise, you will have a problem.
func (p *Pool) Acquire(ctx context.Context) (*grpc.ClientConn, error) {
	var (
		chosen int
		recv   reflect.Value
		ok     bool
	)

main:
	for {
		// get all the cases we want to select on
		cases := p.cases(ctx)

		// if we have more than 2 cases, it means we already have some connections in the pool
		// and we can do Select
		if len(cases) > 2 {
			// we are selecting on multiple channels, we need to get the index of the channel that was selected
			// to decide if it was timeout channel or any other channel with connection
			chosen, recv, ok = reflect.Select(cases)
		} else { // special case when we don't have any connections, and we want directly dial new connection
			// timeout forces us to dial new connection
			// this is optimization when we don't have any connections, and we want to dial new one
			chosen = ChosenAcquireTimeout
		}

		// check on chosen index (0-context, 1-acquireTimeout, 2+ - connection)
		switch chosen {
		case ChosenContextDeadline: // context deadline, check if pool max connections reached

			// safely get number of connections that we provide (not idle, not closed)
			p.mutex.RLock()
			connsLen := len(p.conns)
			p.mutex.RUnlock()

			// if max connections is set, and we reached it, return error
			if p.options.maxConnections > 0 && uint(connsLen) >= p.options.maxConnections {
				return nil, fmt.Errorf("%w: %w", ErrMaxConnectionsReached, ctx.Err())
			}

			// otherwise just return context error
			return nil, ctx.Err()
		case ChosenAcquireTimeout: // this is timeout for acquire connection, so we need to check if there is dialing in progress, and if not, dial new connection
			// try to acquire write lock
			if !p.mutex.TryLock() {
				continue main
			}
			// create new connection with all the bells and whistles
			cc, err := p.createConnection(ctx)
			// lock back
			p.mutex.Unlock()

			if err != nil {
				if errors.Is(err, ErrMaxConnectionsReached) {
					// we reached max connections, so we need to wait for some time and then try again (we will use
					// acquireTimeout for this)
					time.Sleep(p.options.acquireTimeout)
					continue main
				}
				return nil, err
			}
			return cc, err
		default: // we got connection from one of the connection channels

			// we need to handle case when channel was closed (safety reasons)
			if !ok {
				continue main
			}

			// now we know we have client connection, so let's get it from reflect.Value
			cc := recv.Interface().(*grpc.ClientConn)

			// now get the pool connection, so we can update a thing or two
			p.mutex.RLock()
			pc, ok := p.storage[cc]
			p.mutex.RUnlock()
			// if we can't find it, might be some concurrent stealing
			if !ok {
				continue main
			}
			// increment usage given connection
			pc.Usage.Add(1)
			// set last changed to now
			pc.LastChange.Store(ptrTo(time.Now()))

			// return back to caller
			return cc, nil
		}
	}
}

// Close closes the pool, connections and other background resources.
//
// After pool is closed, you cannot do anything with it.
func (p *Pool) Close(ctx context.Context) error {
	if !p.isClosed.CompareAndSwap(true, false) {
		return ErrAlreadyClosed
	}

	// let know all who are listening on close channel that we are closed (goroutines that are cleaning up connections)
	close(p.close)

	// remove connections
	p.mutex.Lock()
	p.conns = nil
	p.mutex.Unlock()

	// close channel
	c := make(chan struct{})

	// check in background if there are no connections left, and if so, close the channel (and thus exit)
	go func() {
		p.cleanupConnections()
		p.mutex.RLock()
		if len(p.storage) == 0 {
			close(c)
		}
		p.mutex.RUnlock()
	}()

	// run cleanup in goroutine eagerly
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c:
			return nil
		}
	}
}

// Forget directly removes connection from the pool.
//
// Warning! This method should be only used when you want the connection to be closed asap.
// For usual use cases, use Release method.
//
// After calling this method, you don't need to call Release.
func (p *Pool) Forget(cc *grpc.ClientConn) error {
	p.mutex.RLock()
	pc, ok := p.storage[cc]
	p.mutex.RUnlock()
	if !ok {
		return ErrInvalidConnection
	}

	// remove now from available connections
	p.mutex.Lock()
	if index := slices.Index(p.conns, pc); index != -1 {
		p.conns = slices.Delete(p.conns, index, index+1)
	}
	p.mutex.Unlock()

	return p.Release(cc)
}

// Release returns a connection to the pool.
//
// It also updates necessary information about the connection (stats, last used time, etc.).
func (p *Pool) Release(conn *grpc.ClientConn) error {
	// release connection
	p.mutex.RLock()
	pc, ok := p.storage[conn]
	p.mutex.RUnlock()

	// connection was not found, some problem?
	if !ok {
		return ErrInvalidConnection
	}

	// if connection has full channel, this means that someone is trying something fishy here
	if pc.isFull() {
		return ErrInvalidConnection
	}

	// check for max lifetime
	if pc.Created.Add(p.options.maxLifetime).Before(time.Now()) {
		// connection should be closed when all are returned
		p.mutex.Lock()
		// check if connection is in the pool
		if index := slices.Index(p.conns, pc); index != -1 {
			p.conns = slices.Delete(p.conns, index, index+1)
		}
		p.mutex.Unlock()
	}

	// add connection back to the channel and update last change
	// this should be enough for pool to work
	pc.ClientConnChan <- conn
	pc.LastChange.Store(ptrTo(time.Now()))

	return nil
}

// Stats returns stats of the pool. It's safe to call this method from multiple goroutines.
// There is corner case when this method can take some time to return. When pool is dialing new connection.
func (p *Pool) Stats() *Stats {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.statsUnsafe()
}

// cases creates a slice of SelectCases that we will use in reflect.Select call.
//
// order of returned SelectCases is following
// 0. - context.Done channel
// 1. - timeout channel
// 1...n - all other channels are channels with grpc connections in order
func (p *Pool) cases(ctx context.Context) []reflect.SelectCase {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// allocate space for all channels + 2 (timeout and context)
	result := make([]reflect.SelectCase, 0, len(p.conns)+2)
	// now we add two additional values
	// * context.Done channel
	// * timeout channel
	// we use these to select on them (wow big revelation)
	result = append(result,
		reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ctx.Done()),
		},
		reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(time.After(p.options.acquireTimeout)),
		},
	)

	// add all channels in order
	for _, conn := range p.conns {
		result = append(result, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(conn.ClientConnChan),
		})
	}

	return result
}

// cleanupConnections checks if any connection should be isClosed.
//
// There are two scenarios which we check
// 1. If connection is idle for longer than maxIdleTime
// 2. If connection is alive for longer than maxLifetime
// 3. If there are connections that are not available anymore (e.g. closed by server)
//
// Warning! this method locks mutex for write (it's understandable why), but it's very fast.
func (p *Pool) cleanupConnections() {
	// we need exclusive access here
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// prepare idle connections
	idleConns := make([]*conn, 0, len(p.conns))

	// iterate over connections and remove those that should be dead by now
	for _, info := range p.storage {

		// check deadline here
		if info.Created.Add(p.options.maxLifetime).Before(time.Now()) {
			if index := slices.Index(p.conns, info); index > -1 {
				p.conns = slices.Delete(p.conns, index, index+1)
				continue
			}
		}

		// check if we also need to handle idle connections
		if p.options.maxIdleTime > 0 {
			// get last changed information to see if it's idle connection
			lastChange := *(info.LastChange.Load())

			maxIdleTime := lastChange.Add(p.options.maxIdleTime)

			// check if it's idle connection and it has all connections released back
			if maxIdleTime.Before(time.Now()) && info.isFull() {
				idleConns = append(idleConns, info)
			}
		}
	}

	// prepare slice for deleted connections
	// these will be deleted at the end of this method
	deletedConns := make([]*grpc.ClientConn, 0)

	// check if we found any idle connections to be closed
	if len(idleConns) > 0 && uint(len(idleConns)) > p.options.maxIdleConnections {
		// now go through idle connections (have in mind max idle connections) and close them right away
		for _, info := range idleConns[p.options.maxIdleConnections:] {
			if index := slices.Index(p.conns, info); index > -1 {
				p.conns = slices.Delete(p.conns, index, index+1)
			}

			deletedConns = append(deletedConns, info.ClientConn)
		}
	}

	// check for connections that are not available anymore but are in map
	{
		// prepare lookup for available connections
		availConns := make(map[*conn]struct{}, len(p.conns))
		for _, conn := range p.conns {
			availConns[conn] = struct{}{}
		}

		// go over connections and close those that are not available anymore and have full channel
		for _, info := range p.storage {
			if _, ok := availConns[info]; ok {
				continue
			}
			if !info.isFull() {
				continue
			}
			deletedConns = append(deletedConns, info.ClientConn)
		}
	}

	// mpw delete all connections that are isClosed
	for _, conn := range deletedConns {
		info, ok := p.storage[conn]
		if !ok {
			p.options.log(context.Background(), "connection is not in map: %v", conn)
			continue
		}

		// we have all connections released back, we should delete connection from map and close it
		if err := info.close(); err != nil {
			p.options.log(context.Background(), "closing connection returned error: %v", err)
		}

		delete(p.storage, info.ClientConn)
	}

}

// createConnection creates new connection, sets connection info to available connections, and returns single
// connection so the caller does not need to wait for it.
//
// this method assumes that mutex is already locked for write access.
func (p *Pool) createConnection(ctx context.Context) (*grpc.ClientConn, error) {
	// check if max connections was set, and if we reached it
	if p.options.maxConnections > 0 && uint(len(p.conns)) >= p.options.maxConnections {
		return nil, ErrMaxConnectionsReached
	}

	// dial new connection
	pc, err := p.dial(ctx, p.statsUnsafe())
	if err != nil {
		return nil, err
	}

	// get one connection so we can satisfy the caller
	cc := <-pc.ClientConnChan

	// increment usage. since we directly return connection to client
	pc.Usage.Add(1)

	// add connection to storage
	p.storage[pc.ClientConn] = pc

	// add connection to available connections, this makes it available for other callers
	p.conns = append(p.conns, pc)

	// return single connection that we prepared for caller
	return cc, nil
}

// dial dials a new connection and returns it
func (p *Pool) dial(ctx context.Context, stats *Stats) (_ *conn, err error) {

	// currently only blocking functions are supported
	opts := make([]grpc.DialOption, 0)
	if !p.options.nonBlocking {
		opts = append(opts, grpc.WithBlock())
	}

	// handle panic in dial
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrDialFailed, r)
		}
	}()

	// first dial the connection
	cc, err := p.options.dialFunc(ctx, stats, opts...)

	// if dialing failed, return error
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDialFailed, err)
	}

	return newConn(cc, p.options), nil
}

// statsUnsafe returns stats of the pool.
//
// this method is private since it assumes that mutex is already locked
// it's used safely and privately in other methods
// if you need to get stats, please use Stats method
func (p *Pool) statsUnsafe() *Stats {
	result := &Stats{
		Connections: make([]ConnStats, 0, len(p.storage)),
	}

	// iterate over all connections and get stats
	for _, info := range p.storage {
		result.Connections = append(result.Connections, info.stats(p.options))
	}

	return result
}
