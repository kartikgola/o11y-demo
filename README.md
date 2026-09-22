# o11y-demo
This repository teaches Observability fundamentals in Go. It is for software engineers.

# Lessons
## 1: Evolution of o11y
- Some history (OpenCensus, OpenTraces, etc)
- Open Standard
- From Silos to Unified Stacks
- The Cost & Complexity Crisis
- Major players (Grafana, Datadog, Elastic, etc)

## 2: The 4 Signals
- Traces & Spans
- Metrics
- Logs
- Profiles

## 3: OTel Client Architecture
- API vs. SDK Separation
- SDK Mechanics
- In-process Context (Go context)
- Cross-process Propagation (W3C Trace Context, B3)
- W3C Baggage & tracestate

## 4: Instrumentation Paradigms
1. Manual instrumentation
    1. Using core SDK and API
    2. Using contrib library
2. Zero-code instrumentation
    1. Compile-time instrumentation
    2. eBPF-based runtime instrumentation (linux only)

## 5: The OpenTelemetry Collector
- Pipeline Building Blocks
- Connectors
- Available Backends & the OTLP Export Matrix
- Sampling Strategies
- Resource Attributes (service.name) & Production Hardening

## 6: Enterprise Fleet & Governance at Scale
- Fleet Control with OpAMP
- Schema Consistency with OTel Weaver
- High-Throughput OTAP (OTel-Arrow)

## 7: Where this is going
- GenAI / LLM agents
- Unified analytics storage

## 8: Ecosystem Snapshot (closing remarks)
- Major vendors (Grafana LG, Datadog, Elastic, Honeycomb, etc)
- OTel maturity (GA signals, Profiles in early dev)
- eBPF (OTel Go Agent, Pyroscope, Parca)
- Frontiers (GenAI, OTAP/Arrow, unified analytics)

# Demo

## Run everything with one command

```
docker compose up -d --build
```

The root `docker-compose.yml` starts the backends, the collector,
Weaver's live-check, and demos 1, 2, 3 and 6 together. Nothing has to be
stopped and restarted between demos.

### One dashboard per demo

Grafana opens with no login and a folder called **o11y demos**. Do not
navigate Explore during the talk — open the dashboard instead:

| Dashboard | URL |
|---|---|
| 00 · Overview — the whole stack | http://localhost:3000/d/o11y-overview |
| 01 · Manual SDK | http://localhost:3000/d/o11y-demo1 |
| 02 · Contrib | http://localhost:3000/d/o11y-demo2 |
| 03 · Compile-time | http://localhost:3000/d/o11y-demo3 |
| 04 · eBPF | http://localhost:3000/d/o11y-demo4 |
| 05 · Weaver | http://localhost:3000/d/o11y-demo5 |
| 06 · Profiles | http://localhost:3000/d/o11y-demo6 |

Add `?kiosk` to any of them to hide the Grafana chrome for presenting.

Each dashboard carries its own explanatory text panels, so the story is
on the screen and you do not have to remember it. The JSON lives in
[demo-lgtm/grafana/dashboards/](demo-lgtm/grafana/dashboards/) and is
provisioned on startup — edit a file, wait 10 seconds, reload.

**There are no log panels.** Loki is running and wired up, but no demo
app emits OTel logs today, so a logs panel would be permanently empty.

| Port | What |
|------|------|
| 3000 | Grafana (no login needed) |
| 8001 | Demo 1 — manual SDK |
| 8002 | Demo 2 — contrib (`otelhttp`) |
| 8003 | Demo 3 — compile-time (`otelc`) |
| 8006 | Demo 6 — profiles |

Demo 4 is **not** in that file. eBPF needs a real Linux kernel with BTF,
so it runs in its own Lima VM. Start it separately, before the talk. See
[demo-4-ebpf/](demo-4-ebpf/).

Each demo folder keeps its own `docker-compose.yml`. Use those to read
one demo in isolation. Use the root file to present.

### Day of the talk — the whole sequence

**T-15 min. Start everything.**

```bash
cd ~/github.com/kartikgola/o11y-demo

# 1. the Mac stack: backends, collector, Weaver, demos 1, 2, 3 and 6
docker compose up -d --build

# 2. the eBPF VM: demo 4 only
limactl start demo4-ebpf
limactl shell demo4-ebpf -- bash -c \
  "cd $(pwd)/demo-4-ebpf && docker compose up -d --build"

# 3. keep demos 1, 2, 3 and 6 busy. Leave this in its own tab for the talk.
#    Without it, their rate() panels in Grafana read 0 — see the note below.
#    Demo 4 is not here: it drives itself from inside the VM.
while true; do
  for p in 8001 8002 8003; do curl -s -o /dev/null localhost:$p/hello; done
  curl -s -o /dev/null localhost:8006/work
done
```

> **Why demo 4 drives itself.** Lima forwards port 8004 to the Mac, but that
> tunnel can die while the socket stays bound. A load loop on the Mac then
> fails silently and the eBPF dashboard goes flat with nothing looking broken.
> So `demo-4-ebpf/docker-compose.yml` runs its own `demo4-load` container next
> to the app. All three of its containers are `restart: unless-stopped`, so
> they come back by themselves after a VM restart.
>
> If `curl localhost:8004/hello` from the Mac ever returns nothing, the
> forward has died, not the demo. `limactl stop demo4-ebpf && limactl start
> demo4-ebpf` rebuilds it, and the containers restart on their own.

> **Why `$(pwd)` and not `~`.** Inside the VM, `~` is the VM's own home
> (`/home/<you>.guest`), which does not hold this repo. Lima mounts your Mac
> home at the same absolute path (`/Users/<you>`), read-only. `$(pwd)` expands
> on the Mac before the command reaches the VM, so run these two commands from
> the repo root. A literal `~` gives you `cd: No such file or directory`.

**T-5 min. Smoke check.**

```bash
docker compose ps                                  # expect 11 running
docker compose logs weaver-livecheck | tail -3     # "will stop after 3600 seconds"

for p in 8001 8002 8003; do curl -s localhost:$p/hello; done
limactl shell demo4-ebpf -- curl -s localhost:8004/hello

open http://localhost:3000                         # admin / admin
```

In Grafana, confirm Tempo lists all 4 services and Pyroscope lists
`demo6-profiles`. If Pyroscope is empty, wait — see the timings below.

**Measured on this machine, with images already pulled:**

| Step | Time |
|------|------|
| `docker compose up -d --build` to 11 containers running | 35 s |
| Weaver pulling the upstream registry | 30–60 s |
| First profile queryable in Pyroscope | 60 s – 2 min, unpredictable |

**Teardown.**

```bash
docker compose down
limactl shell demo4-ebpf -- bash -c \
  "cd $(pwd)/demo-4-ebpf && docker compose down"
limactl stop demo4-ebpf
```

### Start at least 5 minutes early, and leave a load loop running

```
while true; do
  for p in 8001 8002 8003 8004; do curl -s -o /dev/null localhost:$p/hello; done
  curl -s -o /dev/null localhost:8006/work
done
```

Demo 4 runs in the VM, but Lima forwards 8004, so the Mac can drive all
five from one loop.

**Every `rate()` panel reads 0 without this.** The demos serve requests;
none of them generate their own. An idle service has a request rate of
zero, which is correct and looks broken. Start the loop before you open
Grafana.

Traces, metrics and Weaver verdicts all appear within seconds. **Profiles
do not.** Pyroscope takes anywhere from about 10 seconds to 2 minutes to
make a pushed profile queryable, and the delay is not predictable — I
measured 8 s, 37 s, 62 s and over 90 s on the same stack. The capture
itself is fast; the collector sees the samples ~5 s after a burst.

So keep that loop running and present the flame graph as data that is
already there. See [demo-6-profiles/](demo-6-profiles/).

## Prerequisite: Shared backend stack — [demo-lgtm/](demo-lgtm/)
The backend configuration lives here. The root `docker-compose.yml`
mounts these files, so you do not need to start this stack separately.

## 1: Manual instrumentation — [demo-1-manual-sdk/](demo-1-manual-sdk/)
Write every span and every metric by hand with the core OTel SDK.

## 2: Contrib instrumentation — [demo-2-contrib-http/](demo-2-contrib-http/)
Instrument an HTTP handler automatically with the otelhttp contrib library.

## 3: Compile-time instrumentation — [demo-3-compile-time/](demo-3-compile-time/)
Instrument a plain Go app at build time. Write no OTel code.

## 4: Zero-code eBPF instrumentation — [demo-4-ebpf/](demo-4-ebpf/)
Attach to an already-running process from outside with eBPF. Change no code.

## 5: Weaver — [demo-5-weaver/](demo-5-weaver/)
Check live trace attributes against the official semantic-conventions registry.

## 6: Profiles – the 4th signal — [demo-6-profiles/](demo-6-profiles/)
Capture Go CPU profiles and push them to the collector.