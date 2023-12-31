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

// Config is compatible with viper config and mapstructure
//
// It supports default values in struct tags, so you can use it with https://github.com/mcuadros/go-defaults
type Config struct {
	AcquireTimeout     time.Duration `mapstructure:"acquire_timeout" default:"50ms"`
	CleanupInterval    time.Duration `mapstructure:"cleanup_interval" default:"5s"`
	MaxConcurrency     uint          `mapstructure:"max_concurrency" default:"1000"`
	MaxConnections     uint          `mapstructure:"max_connections" default:"0"`
	MaxIdleConnections uint          `mapstructure:"max_idle_connections" default:"0"`
	MaxIdleTime        time.Duration `mapstructure:"max_idle_time" default:"60s"`
	MaxLifetime        time.Duration `mapstructure:"max_lifetime" default:"30m"`
}

// Options returns options by given config
func (c *Config) Options() []Option {
	return []Option{
		WithAcquireTimeout(c.AcquireTimeout),
		WithCleanupInterval(c.CleanupInterval),
		WithMaxConcurrency(c.MaxConcurrency),
		WithMaxConnections(c.MaxConnections),
		WithMaxIdleConnections(c.MaxIdleConnections),
		WithMaxIdleTime(c.MaxIdleTime),
		WithMaxLifetime(c.MaxLifetime),
	}
}
