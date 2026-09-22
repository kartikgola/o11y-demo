# Demo 4: eBPF runtime instrumentation (zero-code, Linux only)

Rung 4, the last rung. Nothing in the app changes — not the source, not
the build, not the image. All instrumentation comes from a sidecar:
[OBI](https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation)
(OpenTelemetry eBPF Instrumentation), the OTel-official successor to
Grafana Beyla. It attaches to the already-running process from outside.

## This demo has no source of its own

The app is [Demo 3's `main.go`](../demo-3-compile-time/main.go), built
with `Dockerfile.plain` — an ordinary `go build`, no `otelc`. That is
deliberate: Demo 3 and Demo 4 instrument the **same source file** two
different ways, so the only thing you are comparing is the technique.

## Why it needs its own VM

eBPF needs a real Linux kernel with BTF, plus `pid: host` and
`privileged: true`. Docker Desktop's own kernel does not qualify. So
this one demo runs in a small Lima VM and reaches the collector on the
Mac over `host.docker.internal:4318`.

**Start it before the talk, not during it.**

### One-time setup
```
limactl start --name=demo4-ebpf template:docker-rootful \
  --cpus=2 --memory=2 --disk=10 --tty=false
```
Rootful Docker, on a real Ubuntu-LTS kernel. Confirmed kernel 6.8.0 with
`/sys/kernel/btf/vmlinux` present. Rootless Docker adds a namespace layer
that breaks the `privileged` + `pid: host` combination eBPF needs.

## What you see
- 1 `SERVER` span per request, named `GET /hello` — the real path, not a
  route template. OBI does not know `http.ServeMux` exists. It only sees
  raw HTTP on the wire.
- **2 extra `INTERNAL` child spans**: `in queue` and `processing`. No
  code-level approach produced these. OBI breaks down time-to-first-byte
  at the OS and runtime level.
- A resource that names its own instrumentation:
  `telemetry.distro.name = opentelemetry-ebpf-instrumentation`.
- **HTTP metrics, too.** Verified on this stack: OBI emits the same three
  histograms `otelhttp` does — `http_server_request_duration_seconds`,
  `http_server_request_body_size_bytes`,
  `http_server_response_body_size_bytes` — 46 timeseries, tagged
  `otel_scope_name="go.opentelemetry.io/obi"`. No extra configuration;
  setting `OTEL_EXPORTER_OTLP_ENDPOINT` is enough.

  An earlier version of this README claimed OBI produced no metrics at
  all. That was wrong against `otel/ebpf-instrument:main` as of
  September 2026. What it still does **not** produce is Go runtime
  metrics — for those you need Demo 3's approach.

## What you do live
The Mac-side stack is already running. See the root
[README](../README.md).

1. Confirm the VM can reach the collector:
   `limactl shell demo4-ebpf -- curl -s -o /dev/null -w '%{http_code}\n' http://host.docker.internal:4318`
   A `404` is fine — that is the collector answering. Connection refused
   means the root stack is not up.
2. `limactl shell demo4-ebpf -- bash -c "cd $(pwd) && docker compose up -d --build"`
   Run this from **this** directory. `$(pwd)` expands on the Mac before the
   command reaches the VM, which is the point — inside the VM, `~` is the
   VM's own home and does not hold this repo.
3. `limactl shell demo4-ebpf -- curl localhost:8004/hello` a few times.
4. `limactl shell demo4-ebpf -- docker logs demo4-obi` — it prints every
   captured request live, because `OTEL_EBPF_TRACE_PRINTER=text`. It
   exports the same spans over OTLP at the same time.
5. On the Mac: open <http://localhost:3000/d/o11y-demo4> and pick a trace to
   see the 3-span breakdown. No Explore navigation needed.

## What you should point out
- The app image is identical to Demo 3's apart from the build command.
  The only new thing anywhere is the `obi` sidecar.
- OBI finds its target by **port** (`OTEL_EBPF_OPEN_PORT: 8004`), not by
  container, image, or language.
- This demo uses `privileged: true` for simplicity, matching OTel's own
  docs. Production should grant only `CAP_BPF`, `CAP_SYS_PTRACE`, and
  `CAP_SYS_ADMIN` (the last for Go context propagation). See OBI's
  [security docs](https://opentelemetry.io/docs/zero-code/obi/security/).

## Teardown
```
limactl shell demo4-ebpf -- bash -c "cd $(pwd) && docker compose down"
limactl stop demo4-ebpf     # keeps the disk image for next time
```
