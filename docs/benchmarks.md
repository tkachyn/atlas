# Benchmark notes

These measurements were collected locally on September 5, 2026 with Go 1.27.0
on Windows.

The server used `127.0.0.1:6391` and a local persistence file. The load
generator used two concurrent clients and 200 total requests per command.

## Results

### `GET`

```text
requests: 200
errors: 0
duration: 342.3331ms
throughput: 584.23 requests/sec
average latency: 3.41297ms
```

### `SET`

```text
requests: 200
errors: 0
duration: 301.0779ms
throughput: 664.28 requests/sec
average latency: 2.986293ms
```

The `GET` run seeds its key before measuring requests. The `SET` run includes
the cost of appending and syncing each command to the local persistence file.

These are basic local measurements, not production capacity claims. Results
depend on hardware, operating system, persistence settings, client count, and
request count.

## Reproduce

Start Atlas:

```powershell
go run ./cmd/atlas
```

Run the load generator:

```powershell
go run ./cmd/atlas-load `
  -command GET `
  -clients 2 `
  -requests 100
```

Replace `GET` with `SET` to measure writes.
