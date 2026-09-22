# Demo 2: Contrib instrumentation (`otelhttp`)

Rung 2 of the ladder. One wrapper call replaces every line of
hand-written tracing from Demo 1.

## What you see
- 1 span per request, named `GET /` — the method and the route. You did
  not write that name.
- **12 attributes** on that span, for example `http.request.method`,
  `url.path`, `http.response.status_code`, `server.address`,
  `network.peer.address`, and `user_agent.original`. `otelhttp` filled in
  every one.
- **3 histograms** per request, with no counter code at all:
  `http_server_request_duration_seconds`,
  `http_server_request_body_size_bytes`,
  `http_server_response_body_size_bytes`.

## What you do live
The stack is already running. See the root [README](../README.md).

1. `curl localhost:8002/hello`
2. Grafana → Explore → Tempo → search `demo2-contrib-http` → open a
   `GET /` span → scroll the attribute list.
3. Grafana → Explore → Prometheus →
   `histogram_quantile(0.95, sum(rate(http_server_request_duration_seconds_bucket[5m])) by (le))`

## What you should point out
- The SDK setup — `TracerProvider`, `MeterProvider`, the OTLP exporters —
  is almost identical to Demo 1. **That step always stays manual.**
- The handler itself has no tracing code. The only OTel call in the
  request path is `otelhttp.NewHandler(mux, "demo2-http-server", ...)`.
- Demo 1 wrote 2 attributes and 1 counter by hand. This handler writes 0
  and 0, and still gets 12 attributes and 3 histograms.
- The trade: you get what the library decided to capture, not exactly
  what you would have chosen.
