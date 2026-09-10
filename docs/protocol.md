# Protocol

Atlas uses a line-oriented request and response protocol over TCP.

Each request is one command followed by a newline. Each response also ends
with a newline.

## Commands

| Command | Arguments | Description |
| --- | --- | --- |
| `PING` | none | Check that the server is responding |
| `SET` | `key value` | Store a value |
| `GET` | `key` | Return a value |
| `DEL` | `key` | Delete a key |
| `EXISTS` | `key` | Check whether a key exists |
| `EXPIRE` | `key seconds` | Set a key lifetime |
| `TTL` | `key` | Return remaining lifetime |

Command names are case-insensitive:

```text
get name
GET name
Get name
```

All three forms are interpreted as `GET name`.

Keys and values cannot contain spaces. Quoting is not currently supported.

Requests are limited to 64 KiB, including the command and its arguments.

`EXPIREAT` is reserved for persistence replay and is not part of the
client-facing command set.

## Examples

### Health check

```text
PING
PONG
```

### Set and retrieve a value

```text
SET name Alex
OK

GET name
Alex
```

### Delete a value

```text
DEL name
1

DEL missing
0
```

### Check existence

```text
EXISTS name
1

EXISTS missing
0
```

### Expire a value

```text
EXPIRE name 30
1

TTL name
29
```

## Missing keys

Missing keys return the following values:

```text
GET      (nil)
DEL      0
EXISTS   0
TTL      -2
```

`TTL` returns `-1` when a key exists without an expiration.

## Errors

Invalid requests return an `ERR` response:

```text
GET
ERR GET requires key
```

Unknown commands are rejected:

```text
PING
ERR unknown command "PING"
```
