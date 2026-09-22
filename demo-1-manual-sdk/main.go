package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otlpmetrichttp "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otlptracehttp "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func main() {
	ctx := context.Background()
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	res := resource.NewWithAttributes(semconv.SchemaURL,
		semconv.ServiceName("demo1-manual-sdk"),
	)

	// 1. Create the trace exporter and the TracerProvider by hand.
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

	// 2. Create the metric exporter and the MeterProvider by hand.
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

	// 3. Create one span and one counter. Set the name and the attributes by hand.
	tracer := otel.Tracer("demo1")
	meter := otel.Meter("demo1")
	requests, _ := meter.Int64Counter("demo1.requests")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reqCtx, span := tracer.Start(r.Context(), "handle_request")
		defer span.End()

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.path", r.URL.Path),
		)
		requests.Add(reqCtx, 1, metric.WithAttributes(
			attribute.String("http.method", r.Method),
		))

		fmt.Fprintf(w, "hello %s\n", r.URL.Path)
	})

	log.Println("demo1 listening on :8001")
	log.Fatal(http.ListenAndServe(":8001", nil))
}
