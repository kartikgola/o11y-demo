# demo-lgtm — backend configuration

This folder holds the **configuration** for the shared backend stack. It
no longer has its own `docker-compose.yml`. The root
[docker-compose.yml](../docker-compose.yml) mounts these files, so start
everything from the repo root:

```
docker compose up -d --build
```

| File | Used by |
|------|---------|
| `collector/config.yaml` | the OTel collector |
| `tempo/tempo.yaml` | Tempo |
| `loki/loki.yaml` | Loki |
| `prometheus/prometheus.yml` | Prometheus |
| `grafana/datasources.yml` | Grafana provisioning |

## Signals → backend → port

| Signal   | Backend    | Host port |
|----------|------------|-----------|
| Traces   | Tempo      | 3200 |
| Metrics  | Prometheus | 9090 |
| Logs     | Loki       | 3100 |
| Profiles | Pyroscope  | 4040 |
| Ingest   | Collector  | 4317 (OTLP gRPC), 4318 (OTLP HTTP), 4319 (pprof push) |
| UI       | Grafana    | 3000 (admin / admin) |

Every demo app sends OTLP to the collector, which fans it out:

- **Traces** → Tempo, and a copy → Weaver's live-check (see
  [demo-5-weaver/](../demo-5-weaver/)).
- **Metrics** → Prometheus, scraped from the collector's `:8889`
  endpoint every **5 seconds**.
- **Logs** → Loki.
- **Profiles** → Pyroscope, either over OTLP or as a pushed
  `runtime/pprof` profile at `:4319/v1/pprof` (see
  [demo-6-profiles/](../demo-6-profiles/)).

The profiles pipeline is alpha upstream and needs the
`service.profilesSupport` feature gate, which the root compose passes.
It works today; a future collector release may break it.

## Adding another app to the stack

Add a service to the root compose file:

```yaml
  your-app:
    build: ./your-app
    environment:
      OTEL_EXPORTER_OTLP_ENDPOINT: "collector:4318"
    networks:
      - o11y-net
    depends_on:
      - collector
```
