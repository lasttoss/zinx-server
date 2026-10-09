# zinx-server

A compact **TCP game server in Go**, built on [zinx](https://github.com/aceld/zinx):
device / Google / Apple / token authentication with JWT, **one live session per account**
enforced through Redis, MongoDB-backed player accounts, chat rooms and an interceptor chain
that guards the authenticated part of the protocol.

[![ci](https://github.com/lasttoss/zinx-server/actions/workflows/ci.yml/badge.svg)](https://github.com/lasttoss/zinx-server/actions/workflows/ci.yml)

```
$ make up
smoke test against 127.0.0.1:8999
  [PASS] ping - pong
  [PASS] auth by device - msgId=1002
  [PASS] auth returns a jwt - token=165 chars
  [PASS] auth returns the account
  [PASS] authorized request (account) - msgId=1100
  [PASS] re-auth on a new connection
OK - the stack is healthy (mongodb + redis + tcp game server)
```

## What this demonstrates

- **Binary protocol design** over raw TCP: an 8-byte TLV frame (`msgId uint32 | length uint32`,
  big endian) with JSON payloads, so a game client can be written in any language in a few lines
  (`scripts/smoke.py` is a full client in ~90 lines of Python).
- **Message-id routing instead of HTTP paths**: ids below `1100` are public (ping, login),
  everything above runs through an interceptor that requires a validated session.
- **Single device per account**: the `connId` of the live session is kept in Redis. When the same
  account authenticates from a second socket, the first one is closed with
  `another device login error` instead of silently forking the player's state.
- **Identity linking**: the same account can be reached by device id, Google id or Apple id, and
  a previously issued JWT can be exchanged for a fresh token.
- **Operational basics**: `/app/healthcheck` binary + container `HEALTHCHECK`, docker compose
  stack with real health-gated startup order, a cron-driven CCU gauge, and a CI pipeline that
  boots the stack and runs an end-to-end smoke test against it.

## Architecture

```mermaid
flowchart LR
  C[Game client<br/>Unity / Cocos / custom] -->|TCP :8999 TLV frames| S[zinx server]
  S --> I{interceptor<br/>msgId >= 1100}
  I -->|no session| R[RpcError + close]
  I -->|ok| H[handlers]
  H --> AS[auth service]
  H --> US[user service]
  AS --> U[(MongoDB users)]
  US --> U
  AS --> RS[(Redis sessions)]
  RS --> I
```

## Protocol

Every frame is `msgId(uint32, big endian) | length(uint32, big endian) | payload(length bytes)`.

| msgId  | name                 | auth | payload                        | response |
|-------:|----------------------|:----:|--------------------------------|----------|
| 1000   | ping                 | no   | anything                       | `pong` (raw) |
| 1001   | auth by token        | no   | `{"id":"<jwt>"}`               | `{"token":...,"data":{...}}` |
| 1002   | auth by device       | no   | `{"id":"<device id>"}`         | same, account created on first call |
| 1003   | auth by Google       | no   | `{"id":"<google id>"}`         | same |
| 1005   | auth by Apple        | no   | `{"id":"<apple id>"}`          | same |
| 1006   | join chat room       | no   | `{"id":"<room>"}`              | room state |
| 1100   | get account          | yes  | `{}`                           | account document |
| 9999   | error                | -    | `{"code":1000,"message":"..."}`| - |

Error codes live in `internal/utils/error_util.go` (`1000+`, plus `9999` for the error frame).

## Quickstart

Requires Docker with the compose plugin.

```bash
git clone https://github.com/lasttoss/zinx-server.git
cd zinx-server
make up          # builds, starts mongodb + redis + server, then runs the smoke test
```

Already using `:6379`, `:27017` or `:8999` for something else? Copy `.env.example` to `.env` and
change `REDIS_PORT`, `MONGO_PORT`, `SERVER_PORT` (`make up` and `make smoke` both read it).

```bash
make logs        # follow the server log
make down        # stop the stack
```

Local development without Docker for the app itself:

```bash
make infra       # mongodb + redis only
make run         # go run ./cmd/server  (config.yaml points at localhost)
make test        # unit tests
```

## Tests

```bash
make race        # go test -race ./...
make coverage    # prints the total coverage line
```

Unit tests cover the JWT lifecycle (subject + expiry, expired token rejected, foreign signature
rejected, garbage rejected) and the api error encoding. The integration path
(ping → auth → authorized request → session enforcement) is exercised by `scripts/smoke.py`,
which CI runs against the compose stack.

## Configuration

`config.yaml` holds development defaults; every key can be overridden from the environment with
the `MYAPP_` prefix (dots become underscores), which is how `docker-compose.yml` points the
server at the `mongodb` / `redis` services.

| config.yaml        | environment              | default |
|--------------------|--------------------------|---------|
| `database.url`     | `MYAPP_DATABASE_URL`     | `mongodb://root:devpassword@localhost:27017` |
| `database.name`    | `MYAPP_DATABASE_NAME`    | `zinx` |
| `redis.url`        | `MYAPP_REDIS_URL`        | `redis://localhost:6379/1` |
| `jwt.secret`       | `MYAPP_JWT_SECRET`       | `dev-secret-change-me` |
| `google.client_id` | `MYAPP_GOOGLE_CLIENT_ID` | empty |
| `apple.client_id`  | `MYAPP_APPLE_CLIENT_ID`  | empty |

`conf/zinx.json` (zinx's own file) holds the listen address, connection limit and max package
size; the port is `8999`.

## Project layout

```
cmd/server          entry point
cmd/healthcheck     tiny TCP probe used by HEALTHCHECK / orchestrators
internal/configs    viper config + mongo/redis clients
internal/routers    route table (msgId -> handler) and the connection lifecycle hooks
internal/filters    interceptor: session check for msgId >= 1100
internal/handlers   one handler per message id
internal/services   auth / user / redis business logic
internal/repositories  mongo access
internal/mappers    wire DTOs, internal/models  documents, internal/constants  RPC ids
scripts/smoke.py    end-to-end client + smoke test
```

## Fixed while preparing this repository

The first version committed here had never actually been run outside a laptop. Auditing it turned
up four real problems, all fixed:

1. **The container was unreachable.** `conf/zinx.json` pinned `Host: 127.0.0.1`, so the server
   bound to loopback *inside* the container: the healthcheck passed while every external
   connection was reset by the port forwarder. Now `0.0.0.0`.
2. **A panic that could take the process down.** In the interceptor, a failed `GetProperty`
   was reported and the connection closed, but execution continued into
   `property.(string)` on a nil value, and the reader goroutine was not the only caller - a
   nil interface type assertion panics. It now returns early through a single `reject` path.
3. **`docker compose up` could not work from a clean clone.** The compose file required a
   gitignored `.env`; the stack now starts with built-in development defaults and documents them
   in `.env.example`.
4. **47 MB of vendored third-party code** (1828 files) was committed; dependencies now come from
   the module cache, and `go.sum` pins them.

## Notes / limitations

- Facebook login (`1004`) is a stub: the route is registered, the handler is empty.
- Session keys in Redis have no TTL yet; a session is cleared on re-authentication and at boot.
- The chat room handler keeps rooms in memory (per process), which is what a single-node server
  needs and is the first thing to replace when you run more than one pod.

## License

MIT - see [LICENSE](LICENSE). Third-party dependencies keep their own licenses (`go.mod`).

## Tests

```bash
go test -race ./...
```

12 tests, and they are where the two real bugs in this repository came from:

- `internal/utils` (60%): JWT signing and verification, expiry, a token signed with the wrong key,
  and the error helper that turns failures into the codes the client sees.
- `internal/services` (10.7%): the chat room driven over real websocket connections - a client
  joining, leaving, leaving twice, a broadcast reaching every client in the room, a broadcast to an
  empty room, and a client whose socket died being dropped instead of being written to forever.
- `internal/handlers` (5.6%): the chat router's room map. The router is registered as an empty
  literal, so the first player to open the chat used to panic the process with *assignment to entry
  in nil map*; and because each connection is handled on its own goroutine, two players joining at
  once were a *concurrent map write*, which Go turns into an unrecoverable crash. Both are fixed
  (lazily created, mutex guarded map) and the -race test fails on the old code.

Coverage is low on purpose and honestly: most of this server is repositories talking to MongoDB and
Redis, and configuration read through viper, which need those services to mean anything. What can
be tested without them is tested, and the whole server is additionally exercised end to end by
`scripts/smoke.py` in CI, against the real MongoDB and Redis in `docker-compose.yml`.
