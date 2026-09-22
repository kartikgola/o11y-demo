# Demo 3: Compile-time instrumentation (zero-code)

Rung 3 of the ladder. `main.go` has **zero** `go.opentelemetry.io/*`
imports and `go.mod` has **zero** OTel dependencies. Only the build
command changes: `otelc go build` instead of `go build`.

`otelc` comes from
[`open-telemetry/opentelemetry-go-compile-instrumentation`](https://github.com/open-telemetry/opentelemetry-go-compile-instrumentation),
the official OTel compile-time tool. Pinned here to **v1.1.0**.

## One source, two demos

This folder holds the plain app that **both** zero-code demos use:

| File | Build | Used by |
|------|-------|---------|
| `Dockerfile` | `otelc go build` | Demo 3 (this one) |
| `Dockerfile.plain` | plain `go build` | [Demo 4](../demo-4-ebpf/), eBPF |

`diff Dockerfile Dockerfile.plain` is the entire difference between
compile-time instrumentation and eBPF instrumentation. The application
source is byte-for-byte the same.

## What you see
- 1 span per request, named `GET /` — the same as Demo 2's `otelhttp`
  span. `otelc` found the plain `net/http.HandleFunc` call by itself.
- The same 12 request attributes Demo 2 gets for free.
- A far richer **resource** than Demo 1 or Demo 2 set: `otelc`
  auto-detects 12 attributes, including `host.name`, `os.type`,
  `os.description`, `process.pid`, `process.owner`,
  `process.runtime.version`, and `process.command_args`. The other two
  demos set only `service.name`.
- Go runtime metrics with no extra code: `go_goroutine_count`,
  `go_memory_used_bytes`, `go_memory_allocated_bytes_total`,
  `go_memory_gc_goal_bytes`, `go_config_gogc_percent`,
  `go_processor_limit`.
- **No HTTP request-duration metric.** `otelc` v1 auto-instruments
  `net/http` for traces only. "Zero-code" means a curated library list,
  not everything.

## What you do live
The stack is already running. See the root [README](../README.md).

1. `curl localhost:8003/hello` a few times.
2. Open `main.go`. Compare it with Demo 1's and Demo 2's. It has 0
   OTel-related lines.
3. Grafana → Explore → Tempo → search `demo3-compile-time` → open the
   `GET /` span → compare its attributes with Demo 2's.
4. Grafana → Explore → Prometheus →
   `go_goroutine_count{exported_job="demo3-compile-time"}`

## What you should point out
- The build command is the *only* change from a plain Go app. No import,
  no wrapper call, no SDK setup.
- This is the newest and least mature of the three code-level
  approaches, and it covers only a curated library list — `net/http`,
  `database/sql`, and a handful of popular packages. It is not the
  universal auto-instrumentation a JVM agent gives you.
- Demo 4 goes one step further and needs no build change at all.

## Two gotchas worth knowing
1. **`OTEL_EXPORTER_OTLP_ENDPOINT` must include a scheme here**
   (`http://collector:4318`). Demos 1 and 2 use a bare `collector:4318`.
   `otelc`'s injected SDK uses standard env-var autoconfiguration, which
   needs a full URL; the hand-written exporters call `.WithEndpoint()`
   and accept `host:port`. Give this one a bare host:port and the SDK
   reads `collector` as the *scheme*, builds `https:///v1/traces`, and
   logs an export error instead of failing to start.
2. **Runtime metrics default to a 60-second export interval.** The root
   compose sets `OTEL_METRIC_EXPORT_INTERVAL: "5000"` so you are not
   standing there for a minute waiting for `go_goroutine_count`.
