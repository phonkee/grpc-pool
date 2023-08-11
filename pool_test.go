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
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"testing"
	"time"
)

// dialDummyGrpc dials a dummy grpc server (actually real one)
// you can alter the behavior of the server by passing functions that will be executed when connection was established
func dialDummyGrpc(t *testing.T, ctx context.Context, fns ...func(conn *grpc.ClientConn)) *grpc.ClientConn {
	listener, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)

	go func() {
		srv := grpc.NewServer()
		// check for close
		go func() {
			<-ctx.Done()
			srv.Stop()
		}()
		assert.Nil(t, srv.Serve(listener))
	}()

	for {
		ctx, cf := context.WithTimeout(context.Background(), 1*time.Second)
		cc, err := grpc.DialContext(ctx, listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		cf()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			assert.NoError(t, err)
		}
		for _, fn := range fns {
			fn(cc)
		}

		return cc
	}
}

// df is a dummy dial function for testing purposes
var df = func(t *testing.T, fns ...func(conn *grpc.ClientConn)) DialFunc {
	return func(ctx context.Context, stats *Stats, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
		return dialDummyGrpc(t, context.Background(), fns...), nil
	}
}

func TestNew(t *testing.T) {
	p, err := New(func(ctx context.Context, stats *Stats, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
		return dialDummyGrpc(t, context.Background()), nil
	})

	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestPool_Acquire(t *testing.T) {
	t.Run("test acquire connection", func(t *testing.T) {
		t.Run("simple test", func(t *testing.T) {
			// check how many times we connected
			called := 0
			// instantiate pool
			p, _ := New(df(t, func(conn *grpc.ClientConn) {
				called += 1
			}), WithMaxConcurrency(2))

			ctx, cf := context.WithTimeout(context.Background(), 24*time.Hour)
			defer cf()

			for i := 0; i < 3; i++ {
				c, err := p.Acquire(ctx)
				assert.NoError(t, err)
				assert.NotNil(t, c)
			}

			assert.Equal(t, 2, called)
			assert.Equal(t, 2, len(p.connMap))

			for i := 0; i < 3; i++ {
				c, err := p.Acquire(ctx)
				assert.NoError(t, err)
				assert.NotNil(t, c)
			}

			assert.Equal(t, 3, called)
			assert.Equal(t, 3, len(p.connMap))
		})

		t.Run("test max connections", func(t *testing.T) {
			p, _ := New(df(t, func(conn *grpc.ClientConn) {}), WithMaxConcurrency(2), WithMaxConnections(1))

			ctx, cf := context.WithTimeout(context.Background(), time.Second)
			defer cf()

			var (
				c   *grpc.ClientConn
				err error
			)

			// acquire more connections than available
			for i := 0; i < 3; i++ {
				c, err = p.Acquire(ctx)
			}

			assert.ErrorIs(t, err, context.DeadlineExceeded)
			assert.ErrorContains(t, err, ErrMaxConnectionsReached.Error())
			assert.Nil(t, c)
		})

		t.Run("test idle connections", func(t *testing.T) {
			t.Run("test max idle connections", func(t *testing.T) {
				p, _ := New(
					df(t, func(conn *grpc.ClientConn) {}),
					WithMaxConcurrency(2),
					WithMaxIdleConnections(2),
					WithMaxLifetime(time.Hour),
					WithMaxIdleTime(time.Second),
					WithCleanupInterval(time.Hour),
				)

				conns := make([]*grpc.ClientConn, 0)
				for i := 0; i < 100; i++ {
					ctx, cf := context.WithTimeout(context.Background(), time.Second)
					c, _ := p.Acquire(ctx)
					cf()
					conns = append(conns, c)
				}
				for _, c := range conns {
					assert.NoError(t, p.Release(c))
				}

				// sleep for a second to let the cleanup goroutine do its job
				time.Sleep(time.Second * 1)

				// now do cleanup
				p.cleanupConnections()

				assert.Equal(t, 2, len(p.connMap))
			})
		})

		t.Run("test forget", func(t *testing.T) {
			p, _ := New(
				df(t, func(conn *grpc.ClientConn) {}),
				WithMaxConcurrency(2),
			)

			var c *grpc.ClientConn
			for i := 0; i < 3; i++ {
				ctx, cf := context.WithTimeout(context.Background(), time.Second)
				c, _ = p.Acquire(ctx)
				cf()
			}

			assert.Equal(t, 2, len(p.connMap))

			_ = p.Forget(c)
			p.cleanupConnections()

			assert.Equal(t, 1, len(p.connMap))
		})

	})
}
