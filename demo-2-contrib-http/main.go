package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	otlpmetrichttp "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otlptracehttp "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func main() {
	ctx := context.Background()
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	res := resource.NewWithAttributes(semconv.SchemaURL,
		semconv.ServiceName("demo2-contrib-http"),
	)

	// 1. Create the trace exporter and the TracerProvider. This step is the
	// same as in Demo 1.
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("trace exporter: %v", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(ctx)

	// 2. Create the metric exporter and the MeterProvider. This step is the
	// same as in Demo 1.
	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("metric exporter: %v", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter,
			sdkmetric.WithInterval(5*time.Second),
		)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	defer mp.Shutdown(ctx)

	// 3. Write the plain business handler. It has no span, no attribute, and
	// no counter.
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello %s\n", r.URL.Path)
	})

	// 4. Wrap the handler once with otelhttp. It creates the span and the
	// HTTP metrics for every request. It also adds the default
	// semantic-convention attributes.
	handler := otelhttp.NewHandler(mux, "demo2-http-server",
		otelhttp.WithTracerProvider(tp),
		otelhttp.WithMeterProvider(mp),
	)

	log.Println("demo2 listening on :8002")
	log.Fatal(http.ListenAndServe(":8002", handler))
}
