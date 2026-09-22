# Demo 1: Manual instrumentation (core SDK)

Rung 1 of the ladder. You write every span, every attribute, and every
metric by hand.

## What you see
- 1 span per request. The span name is `handle_request`, because you
  typed it.
- 2 span attributes, `http.method` and `http.path`, because you typed
  those too.
- 1 metric, the `demo1.requests` counter.
- The collector sends both to Tempo and Prometheus.

## What you do live
The stack is already running. See the root [README](../README.md).

1. `curl localhost:8001/hello`
2. Grafana → Explore → Tempo → search `handle_request`
3. Grafana → Explore → Prometheus →
   `sum by (http_method) (demo1_requests_total)`

## What you should point out
- You wrote the span name, every attribute, and the counter name by hand.
- If you miss one attribute, it is gone. The SDK adds no defaults.
- This file is pinned to semconv **v1.26.0**, so it emits the old
  `http.method` and `http.path`. Demo 5 catches exactly that.
- Compare this with Demo 2, where `otelhttp` does all of it for you.
