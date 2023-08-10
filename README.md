# grpc_pool

Pool of gRPC connections. This pool is somehow special. It does not have single connection for single call,
but it shares single connection for multiple concurrent calls. This is useful when you don't want to overload
your servers with too many gRPC method calls.
Main idea of this pool is that connections are shared concurrently.

This pool is self balancing, so you just need to configure it and it will do the rest.
You can also access pool statistics and expose them to your monitoring system.

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
    WithMaxConcurrentCalls(1000),
    WithMaxIdleConnections(5),
    WithMaxIdleTime(time.Second*10),
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


# author

Peter Vrba <phonkee@phonkee.eu>