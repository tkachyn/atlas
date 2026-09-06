# Persistence

Atlas persists successful commands in an append-only file. The default path is
`atlas.aof`, and the path can be changed with `-data-file`.

## Startup recovery

Atlas opens and replays the persistence file before it starts accepting TCP
connections:

```text
open log
    |
read records in order
    |
apply commands to a new store
    |
start server
```

Replaying commands reconstructs the in-memory state without serializing the
entire map after every operation.

## Record formats

Atlas accepts legacy plain-text records:

```text
SET name Alex
SET age 19
```

New records use a version marker and CRC32 checksum:

```text
A1|checksum|SET name Alex
```

The checksum detects corruption in complete records. Corruption causes startup
to fail instead of being silently ignored.

## Crash recovery

If the file ends in an incomplete record, Atlas truncates that final partial
record and starts normally. This handles a process terminating while a record
is being written.

An invalid complete record is treated as corruption and stops recovery.

## Write ordering

For each client command:

1. Atlas parses and validates the request.
2. Atlas appends the command to the log.
3. Atlas calls `Sync` on the file.
4. Atlas updates the in-memory store.
5. Atlas sends the response.

This ordering avoids acknowledging a command before its log record is flushed.

## Expiration

Client-facing `EXPIRE` commands are stored as absolute expiration timestamps.
This prevents a restart from resetting a key to its original full lifetime.

## Compaction

The log is compacted after reaching the `-max-log-bytes` threshold. Set the
threshold to `0` to disable automatic compaction.

Compaction:

1. Collects the current live store entries.
2. Writes them to a temporary checksummed log.
3. Flushes the temporary file.
4. Replaces the old log.
5. Reopens the replacement for appends.

The resulting log contains only the current state rather than every historical
command.
