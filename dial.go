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
	"google.golang.org/grpc"
)

// DialFunc is a function that dials a gRPC connection.
// This function is passed as required argument to New.
//
// You need to provide your own implementation of this function.
//
// It adds stats information to have context about already established connections.
// This is for case when you need to do more advanced client side load balancing based on already connected connections
type DialFunc func(ctx context.Context, stats *Stats, opts ...grpc.DialOption) (*grpc.ClientConn, error)

// StaticHostDialFunc returns DialFunc that always connects to the same host.
func StaticHostDialFunc(address string, options ...grpc.DialOption) DialFunc {
	return func(ctx context.Context, stats *Stats, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
		dialOpts := make([]grpc.DialOption, 0, len(options)+len(opts))
		dialOpts = append(dialOpts, options...)
		dialOpts = append(dialOpts, opts...)
		return grpc.DialContext(ctx, address, dialOpts...)
	}
}
