# Demo 5: Schema consistency with OTel Weaver

This folder has no code and no compose file of its own. The
`weaver-livecheck` service lives in the root
[docker-compose.yml](../docker-compose.yml), so it is already running
alongside everything else.

The hook is real, not staged. The demos in this repo already disagree
with each other about attribute names for the same thing:

- **Demo 1** is pinned to semconv **v1.26.0** and emits `http.method`
  and `http.path`.
- **Demos 2, 3 and 4** use newer libraries and emit
  `http.request.method` and `url.path`.

[Weaver](https://github.com/open-telemetry/weaver)'s `live-check` exists
to catch this. By default it checks live OTLP traffic against the
**actual upstream** semantic-conventions registry, pulled from GitHub at
startup. No custom registry needed.

## What you do live
The stack is already running. See the root [README](../README.md).

1. `curl localhost:8001/hello` a few times, then
   `curl localhost:8002/hello` a few times.
2. `docker compose logs weaver-livecheck`

Two spans, one HTTP request each, the same registry, two verdicts.

## Real output, captured from a real run

Demo 1's span:
```
Span handle_request `internal`
    http.method = GET
        - [violation] Attribute 'http.method' is deprecated; reason = 'renamed',
          note = 'Replaced by `http.request.method`.'.
    http.path = /hello
        - [violation] Attribute 'http.path' does not exist in the registry.
```

Demo 2's span, same moment, same registry:
```
Span GET / `server`
    server.address = localhost
    http.request.method = GET
    url.scheme = http
    ...
    http.response.status_code = 200
```
Clean. No violations.

## What you should point out
- Nothing changed in Demo 1 or Demo 2 for this. Weaver just watches a
  copy of the trace stream they already send to the collector.
- Weaver checks the **current** upstream registry. This is not a
  snapshot-in-time opinion. Rename an attribute upstream tomorrow and
  this same check flags it the same day.

## Three gotchas hit and fixed
1. **Weaver's OTLP gRPC listener binds to `127.0.0.1` by default,** so no
   other container can reach it. Fixed with
   `--otlp-grpc-address=0.0.0.0`.
2. **Weaver's OTLP receiver rejects gzip** (`Unimplemented desc =
   Content is compressed with 'gzip' which isn't supported`). The
   collector's `otlp` gRPC exporter compresses by default. Fixed with
   `compression: none` on the `otlp/weaver` exporter in
   [`demo-lgtm/collector/config.yaml`](../demo-lgtm/collector/config.yaml).
3. **The collector must start after `weaver-livecheck`.** Start it first
   and its gRPC client resolves an empty address, never retries the
   lookup, and every trace to Weaver fails with `no children to pick
   from` while both containers look perfectly healthy.

   This used to need a manual `docker compose restart collector`. The
   root compose now declares `depends_on: weaver-livecheck` on the
   collector, which fixes the ordering. **The restart step is gone.**

Note the `--inactivity-timeout=3600` in the root compose. Weaver runs a
bounded session and exits when it goes quiet; at the old 300 s it died
part-way through the slides.
