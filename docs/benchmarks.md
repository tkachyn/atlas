# Benchmark notes

These measurements were collected locally on September 9, 2026 with Go 1.27.0
on Windows.

The server ran locally with a local persistence file. The load generator used
two concurrent clients and 200 total requests per command.

## Results

### `GET`

```text
requests: 200
errors: 0
duration: 6.5031ms
throughput: 30754.56 requests/sec
average latency: 52.509µs
```

### `SET`

```text
requests: 200
errors: 0
duration: 345.352ms
throughput: 579.12 requests/sec
average latency: 3.405374ms
```

The `GET` run seeds its key before measuring requests. The `SET` run includes
the cost of appending and syncing each command to the local persistence file.
Read-only commands are not written to the persistence log.

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
