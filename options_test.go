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
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestOptions(t *testing.T) {
	t.Run("test nil dial function", func(t *testing.T) {
		_, err := newOptions(nil)
		assert.ErrorIs(t, ErrInvalidDialFunc, err)
	})
	t.Run("test WithAcquireTimeout", func(t *testing.T) {
		o, _ := newOptions(df(t))
		assert.NoError(t, o.apply(WithAcquireTimeout(42*time.Second)))
		assert.Equal(t, 42*time.Second, o.acquireTimeout)
		assert.ErrorIs(t, ErrInvalidAcquireTimeout, o.apply(WithAcquireTimeout(0)))
	})
	t.Run("test WithCleanupInterval", func(t *testing.T) {
		o, _ := newOptions(df(t))
		assert.NoError(t, o.apply(WithCleanupInterval(time.Second)))
		assert.ErrorIs(t, ErrInvalidCleanupInterval, o.apply(WithCleanupInterval(0)))
	})
	t.Run("test WithLogger", func(t *testing.T) {
		l := LoggerFunc(func(ctx context.Context, msg string) {})
		o, _ := newOptions(df(t))
		assert.NoError(t, o.apply(WithLogger(l)))
		assert.Equal(t, fmt.Sprintf("%p", l), fmt.Sprintf("%p", o.logger))
	})
	t.Run("test WithMaxConcurrency", func(t *testing.T) {
		o, _ := newOptions(df(t))
		assert.NoError(t, o.apply(WithMaxConcurrency(10)))
		assert.Equal(t, uint(10), o.maxConcurrency)
		assert.ErrorIs(t, ErrInvalidMaxConcurrency, o.apply(WithMaxConcurrency(0)))
	})
	t.Run("test WithMaxConnections", func(t *testing.T) {
		o, _ := newOptions(df(t))
		assert.NoError(t, o.apply(WithMaxConnections(10)))
		assert.Equal(t, uint(10), o.maxConnections)
		assert.NoError(t, o.apply(WithMaxConnections(0)))
		assert.Equal(t, uint(0), o.maxConnections)
	})
	t.Run("test WithMaxIdleConnections", func(t *testing.T) {
		o, _ := newOptions(df(t))
		assert.NoError(t, o.apply(WithMaxIdleConnections(10)))
		assert.Equal(t, uint(10), o.maxIdleConnections)
		assert.NoError(t, o.apply(WithMaxIdleConnections(0)))
		assert.Equal(t, uint(0), o.maxIdleConnections)
	})
	t.Run("test WithMaxIdleTime", func(t *testing.T) {
		o, _ := newOptions(df(t))
		assert.NoError(t, o.apply(WithMaxIdleTime(42+time.Second)))
		assert.Equal(t, 42+time.Second, o.maxIdleTime)
		assert.ErrorIs(t, ErrInvalidMaxIdleTime, o.apply(WithMaxIdleTime(0)))
	})
	t.Run("test WithMaxLifetime", func(t *testing.T) {
		o, _ := newOptions(df(t))
		assert.NoError(t, o.apply(WithMaxLifetime(42+time.Second)))
		assert.Equal(t, 42+time.Second, o.maxLifetime)
		assert.ErrorIs(t, ErrInvalidMaxLifetime, o.apply(WithMaxLifetime(0)))
	})
}
