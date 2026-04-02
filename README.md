<p align="center">
 <img src="assets/banner.png" width="350">
</p>

<div align="center">

  <a href="">![Tests](https://github.com/leodeim/cof/actions/workflows/go.yml/badge.svg)</a>
  <a href="">![Code Scanning](https://github.com/leodeim/cof/actions/workflows/codeql.yml/badge.svg)</a>
  <a href="https://codecov.io/gh/leodeim/cof" > 
    <img src="https://codecov.io/gh/leodeim/cof/branch/master/graph/badge.svg?token=3275GV3OGX"/> 
  </a>
  <a href="">![Report](https://goreportcard.com/badge/github.com/leodeim/cof)</a>
  <a href="">![Release](https://badgen.net/github/release/leodeim/cof)</a>
  <a href="">![Releases](https://badgen.net/github/releases/leodeim/cof)</a>
  
</div>

# cof

A lightweight, generic, thread-safe in-memory key-value cache for Go with TTL-based expiration and automatic background cleanup.

## Features

- **Generic** -- store any value type via Go generics
- **Thread-safe** -- all operations are protected by a `sync.RWMutex`
- **TTL expiration** -- default TTL per cache, per-item override with `PutWithTTL`
- **Automatic cleanup** -- background goroutine periodically removes expired entries
- **Lazy expiration** -- `Get`, `Pop`, `Has`, `Len`, and `Keys` never return expired items even between cleanups

## Install

```
go get github.com/leodeim/cof
```

Requires Go 1.23+.

## Quick start

```go
package main

import (
    "fmt"
    "time"

    "github.com/leodeim/cof"
)

func main() {
    c, err := cof.Init[string](
        cof.TTL(5 * time.Minute),
        cof.CleanInterval(1 * time.Minute),
    )
    if err != nil {
        panic(err)
    }
    defer c.Stop()

    c.Put("greeting", "hello, world!")

    v, ok := c.Get("greeting")
    fmt.Println(v, ok) // hello, world! true
}
```

## API

### Creating a cache

```go
c, err := cof.Init[ValueType](opts ...cof.Option)
```

#### Options

| Option | Default | Description |
|---|---|---|
| `cof.TTL(d)` | 15 min | Default TTL for entries. Pass `0` to disable expiration. |
| `cof.CleanInterval(d)` | 1 min | How often the cleaner runs. Pass `0` to disable background cleanup. |

### Operations

| Method | Description |
|---|---|
| `Put(key, value)` | Insert/update with the default TTL |
| `PutWithTTL(key, value, ttl)` | Insert/update with a custom TTL (`0` = no expiry) |
| `Get(key) (T, bool)` | Retrieve a value; returns false if missing or expired |
| `Pop(key) (T, bool)` | Retrieve and delete; returns false if missing or expired |
| `Delete(key)` | Remove an entry |
| `Has(key) bool` | Check existence (expired entries return false) |
| `Len() int` | Count of live entries |
| `Keys() []string` | Sorted slice of live keys |
| `Clear()` | Remove all entries (keeps the cleaner running) |
| `Stop()` | Remove all entries and stop the background goroutine |

## License

[MIT](LICENSE)
