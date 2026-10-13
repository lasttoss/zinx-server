# Measurements

The README carried three numbers that no longer matched the suite. They are corrected here, and this
file is where the corrected ones come from.

## Environment

| | |
|---|---|
| Machine | 12-core x86-64 (AMD64), Ubuntu 24.04, Docker 29.1.3 |
| Toolchain | Go 1.27.2 / Node 24.19.0 / JDK 17 in `maven:3.9-eclipse-temurin-17` |
| Date | 2026-10-13 (UTC) |


## Tests and coverage

```
$ go test ./... -count=1 -v | grep -c '^--- PASS: Test'
50

$ go test ./... -cover
ok  zinx-server/internal/configs        21.7%
ok  zinx-server/internal/filters        95.7%
ok  zinx-server/internal/handlers       52.9%
ok  zinx-server/internal/mappers       100.0%
ok  zinx-server/internal/models        100.0%
ok  zinx-server/internal/services       74.8%
ok  zinx-server/internal/utils          60.0%
    zinx-server/internal/repositories    0.0%
    zinx-server/cmd/server               0.0%
```

**Result.** The suite has grown to **50 tests in 11 files**, from the 12 the README described, and two
of the per-package numbers moved with it:

| README said | measured |
|---|---|
| 12 tests | **50** |
| `internal/services` 10.7% | **74.8%** (the chat-room tests were added after that line) |
| `internal/handlers` 5.6% | **52.9%** |
| `internal/utils` 60% | 60.0% (unchanged) |

**What is not measured here.** The smoke test - the one that talks the raw TLV protocol to a running
server - needs MongoDB and Redis, and it runs in CI against the compose stack. It is the reason the
protocol itself is tested rather than a Go client's view of it.
