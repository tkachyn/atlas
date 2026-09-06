# Architecture

Atlas is organized as a small pipeline. Each layer has one primary
responsibility:

```text
TCP client
    |
TCP server
    |
line-oriented protocol parser
    |
command executor
    |
concurrent in-memory store
    |
append-only persistence log
```

## Request lifecycle

For a request such as `SET name Alex`:

1. The server reads a complete line from the TCP connection.
2. The protocol package validates the command and its arguments.
3. The command package applies the operation to the store.
4. The persistence package records and syncs the command.
5. The server writes a newline-terminated response.

The persistence record is written before the response is sent. This prevents
Atlas from acknowledging a command that was not written to disk.

## Packages

### `cmd/atlas`

Parses command-line settings, opens the persistence file, replays saved
commands, and starts the server.

### `internal/server`

Owns the TCP listener, client lifecycle, request loop, and graceful shutdown.
Each client is handled by a separate goroutine.

### `internal/protocol`

Parses the line-based command format and validates command names and argument
counts.

### `internal/command`

Maps parsed commands to store operations and formats their responses.

### `internal/store`

Stores values, expiration timestamps, and concurrency state. Expiration is
lazy: key operations remove entries whose expiration time has passed.

### `internal/persistence`

Appends commands, verifies checksums during replay, recovers incomplete final
records, and compacts the log into the current live state.

## Concurrency

The server accepts clients in its main listener loop and assigns each client a
goroutine. The store protects its map with a mutex. The persistence log uses a
separate mutex to serialize file writes.

During shutdown, Atlas closes the listener, closes active connections, and
waits for client goroutines to finish.

## Recovery

Startup follows this sequence:

```text
open persistence file
    |
read legacy or checksummed records
    |
apply records to a new store
    |
truncate an incomplete final record
    |
start the TCP listener
```

Expiration commands are stored as absolute timestamps so restarting Atlas does
not reset their remaining lifetime.
