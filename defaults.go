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

// if these are changed, please change also config values in config.go
const (
	// DefaultAcquireTimeout is the default timeout for acquiring a connection from the pool using reflect.Select
	DefaultAcquireTimeout = 50 * time.Millisecond
	// DefaultCleanupInterval is the default interval for cleaning up idle connections and connections that passed their max lifetime
	DefaultCleanupInterval = 5 * time.Second
	// DefaultMaxConcurrency is the default maximum number of concurrent connections
	DefaultMaxConcurrency = 1000
	// DefaultMaxIdleTime is the default maximum time to mark connection as idle when it wasn't used
	DefaultMaxIdleTime = time.Second * 60
	// DefaultMaxLifetime is the default maximum lifetime of a connection
	DefaultMaxLifetime = 30 * time.Minute
)
