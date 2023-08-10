# grpc_pool

High performance gRPC pool for client connections. It's not as usual pool, it does not have single connection for single call,
but rather it shares single connection for multiple concurrent calls. This is useful when you don't want to overload
your servers with too many gRPC method calls on single connection.


This pool is self-balancing, so you just need to configure it, and it will do the rest.

# example

Let me show you example of how to use this pool.

```go
// create new pool
pool, err := grpc_pool.New(
    // Dial function is used to create new connection
    func(ctx context.Context, stats *grpc_pool.PoolStats, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
        // add additional dial options
        opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		// create new connection (always pass options from grpc-pool)
        return grpc.DialContext(ctx, "localhost:50051", opts...) 
    }, 
    // WithMaxConcurrentCalls sets how many concurrent calls can be made on single connection 
    WithMaxConcurrentCalls(1000),
    // WithMaxIdleConnections sets how many idle connections can be kept in pool
    WithMaxIdleConnections(5),
    // WithMaxIdleTime sets after how much time idle connection is marked as idle
    WithMaxIdleTime(time.Second*10),
    // WithMaxConnections sets how many connections can be kept in pool
    WithMaxConnections(20),
)

// prepare context with some timeout
ctx, cf := context.WithTimeout(context.Background(), time.Second*10)
defer cf()

// get connection
conn, err := pool.Acquire(ctx)
if err != nil {
	panic(err)
}

// don't forget to return connection back to pool, otherwise you will leak connections, and pool will be confused.
defer pool.Release(conn)
```


# stats

gRPC pool provides stats about pool. You can use it to monitor your pool.

```go
stats := pool.Stats()
````


# author

Peter Vrba <phonkee@phonkee.eu>