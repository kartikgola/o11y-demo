# Demo 6: Profiles — the 4th signal

Go has no native OTel profiling SDK yet. The profiles signal is still
alpha end-to-end, as confirmed in OTel's own 2026 blog posts. Today's
supported path is the stdlib `runtime/pprof` profile format, pushed to
the collector's `pprof` receiver in `server` mode. This mode already
ships in `otel/opentelemetry-collector-contrib:0.161.0`, the same image
`demo-lgtm` pins. The receiver feeds the same `profiles` pipeline
(`exporters: [otlp/pyroscope]`) that has sat in `demo-lgtm` since it was
built. `main.go` has **zero** third-party dependencies. It uses only
`runtime/pprof` and `net/http`.

## What it does
- `GET /` returns `hello <path>`, the same as every other demo.
- `GET /work` runs 2,000,000 rounds of SHA-256, wrapped in
  `pprof.Do(ctx, pprof.Labels("trace_id", id), ...)`. This load is
  deliberately CPU-heavy, so a captured profile has a clear hot function.
  The label attaches a random hex ID to every CPU sample taken while the
  request runs, and the response echoes it back as `trace_id=<id>`. A
  real OTel-instrumented app would read this ID from the active span
  instead of generating one — the label is what actually correlates a
  profile sample to one request; a real trace ID would just be a more
  useful value to put in it.
- A background goroutine captures 5-second CPU profiles **back to back**,
  with no idle gap. It `POST`s each one to
  `http://collector:4319/v1/pprof` with `Content-Encoding: gzip`. No
  extra encoding step is needed, because Go's `profile.proto` output is
  already gzip-compressed. The loop and the server both stop on
  `SIGINT`/`SIGTERM` through a shared `context.Context`, instead of
  blocking forever on `time.Sleep`.

## A real gotcha hit and fixed
The `pprof` receiver's `server` (push) mode has **no field for resource
attributes**. This was confirmed by reading the `ServerConfig` struct
directly. Every pushed profile landed in Pyroscope labeled
`service_name = "unknown_service"`, confirmed by querying Pyroscope's own
API before and after the fix. The fix lives in
`demo-lgtm/collector/config.yaml`: a `resource` processor, scoped to the
`profiles` pipeline, that upserts `service.name: demo6-profiles` onto
everything the `pprof` receiver produces:
```yaml
processors:
  resource/pprof_service_name:
    attributes:
      - key: service.name
        value: demo6-profiles
        action: upsert
```
This is a small, real echo of Lesson 5's "Resource Attributes
(`service.name`) & Production Hardening" bullet. The collector, not the
app, gives this profile its real name. If a future demo also pushes
profiles this way, it gets the same name, unless you add a second
`pprof` receiver on a different port. This is a documented limitation,
not a bug.

## What you do live
1. From the repo root, `docker compose up -d --build`. This starts the
   backends, the collector and every demo app at once. Run it once,
   before the talk.
2. Run `for i in $(seq 20); do curl -s localhost:8006/work; done`. Note
   one `trace_id` from the output.
3. Watch `docker logs demo6` for `pushed profile: 204 No Content`. A push
   goes out every ~5 seconds.
4. In Grafana, go to Explore → Pyroscope datasource → service
   `demo6-profiles` → profile type `process_cpu` → flame graph.
   `crunchNumbers` should be the hot frame.

## Do not demo this on a tight clock

**The capture is fast. Pyroscope's ingest-to-query path is not, and it is
not predictable.**

The collector consistently sees the samples about 5 seconds after a
burst — that part is reliable. Getting from there to a queryable flame
graph is not. Measured on this stack, same 20-request burst every time:

| Pyroscope state | Visible after |
|---|---|
| Cold, first profile for the service | 62 s, then >90 s on a repeat |
| Warm | 8 s, then 37 s, then >60 s |

So: somewhere between about 10 seconds and 2 minutes, with no way to
predict which you get. This is Pyroscope's segment-writer and metastore,
not the demo.

**What to do instead of waiting on stage:**

1. Start the stack and run a load loop **at least 5 minutes before** you
   present, so the flame graph is already populated:
   ```
   while true; do curl -s localhost:8006/work > /dev/null; done
   ```
2. Leave that loop running during the talk. Fresh data keeps arriving,
   so you show a live flame graph instead of watching a spinner.
3. Present the flame graph as already-collected data. Do not promise the
   room a live round-trip from `curl` to flame graph.

If you want to show the round-trip anyway, show `docker logs demo6`
(`pushed profile: 204 No Content` every ~5 s) as the proof that capture
and push are immediate, and treat the Grafana view as the slow part.
5. Try to filter the flame graph down to the one `trace_id` you noted in
   step 2, with a tag query such as `{trace_id="<id>"}`. **Verify this
   works during rehearsal, before the live talk** — it depends on
   Pyroscope surfacing pprof sample labels as queryable tags for a
   pushed (not pulled) profile, which has not been confirmed yet.

## A second gotcha, found while rehearsing this
The capture loop used to sleep 10 seconds between 5-second captures. If
your `/work` burst landed in that gap, the next profile carried **0
samples**. The collector accepted it, returned `204`, forwarded it, and
logged no error — and nothing at all appeared in Pyroscope. The debug
exporter showed the truth: `"resource profiles": 1, "sample records": 0`.
The loop now captures back to back, so a window is always open.

## What you should point out
- This is the least mature signal of the 4, and the code shows it. This
  is the only demo with hand-rolled push logic instead of an SDK or
  automatic instrumentation. Profiles support has not caught up yet.
- The collector needed a fix for missing resource attributes here.
  Every other signal in this repo carries `service.name` from the app
  itself.
- The `trace_id` label on `/work` is the actual pitch for profiles as a
  4th signal: a profile sample that carries a trace ID lets you jump
  from a slow trace straight to the code that made it slow. Everything
  else in this demo is just "capture and push a profile," which
  standalone profilers have done for years, with no OTel involved.
