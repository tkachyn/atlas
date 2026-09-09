# Atlas

Atlas is a Redis-inspired, in-memory key-value server written in Go. It
provides a small line-based TCP protocol, concurrent client handling,
expiration, and append-only persistence.

## Features

- TCP server with one goroutine per client
- `SET`, `GET`, `DEL`, `EXISTS`, `EXPIRE`, and `TTL`
- Mutex-protected in-memory storage
- Lazy key expiration
- Append-only persistence with replay
- Checksummed records and partial-tail recovery
- Persistence compaction at a configurable size threshold
- Standard-library load generator

## Requirements

- Go 1.27 or newer

## Quick start

Start Atlas with its default configuration:

```powershell
go run ./cmd/atlas
```

Atlas listens on `:6379` and writes persistence records to `atlas.aof`.

To select a different address or persistence file:

```powershell
go run ./cmd/atlas `
  -addr 127.0.0.1:6380 `
  -data-file data/atlas.aof
```

The default compaction threshold is 64 MiB. Set a custom threshold with:

```powershell
go run ./cmd/atlas -max-log-bytes 1048576
```

Set `-max-log-bytes 0` to disable automatic compaction.

## Protocol

Atlas accepts one command per line over TCP:

```text
SET name Alex
OK

GET name
Alex

EXPIRE name 30
1

TTL name
29

DEL name
1
```

Command names are case-insensitive. Keys and values cannot contain spaces in
the current protocol.

See [docs/protocol.md](docs/protocol.md) for command syntax and response
semantics.

## Persistence

Atlas opens and replays its persistence file before accepting clients. A
successful state-changing command is written and synced before Atlas sends its
response.

Legacy plain-text records remain readable. New records include a version marker
and CRC32 checksum. An incomplete final record is truncated during recovery,
while corruption in a complete record stops startup.

The log is compacted when it reaches the configured size threshold. Compaction
writes the current live state to a temporary file and replaces the active log.

See [docs/persistence.md](docs/persistence.md) for the persistence format and
recovery behavior.

## Load measurements

Start Atlas in one terminal:

```powershell
go run ./cmd/atlas
```

Run the load generator in another:

```powershell
go run ./cmd/atlas-load `
  -command GET `
  -clients 10 `
  -requests 1000
```

Use `-command SET` to measure writes. The tool reports request count, errors,
throughput, and average latency.

Recorded local measurements are available in
[docs/benchmarks.md](docs/benchmarks.md).

## Architecture

```text
TCP client
    |
server connection handler
    |
protocol parser
    |
command executor
    |
concurrent store
    |
append-only persistence log
```

See [docs/architecture.md](docs/architecture.md) for package responsibilities
and request flow.

## License

Atlas is available under the [MIT License](LICENSE).
