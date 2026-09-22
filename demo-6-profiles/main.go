package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
)

// crunchNumbers uses heavy CPU load on purpose. This gives a captured
// profile a clear hot function for the flame graph.
func crunchNumbers() {
	sum := sha256.Sum256([]byte("demo6"))
	for i := 0; i < 2_000_000; i++ {
		sum = sha256.Sum256(sum[:])
	}
}

// newTraceID returns a random hex ID for pprof-label correlation. A real
// OTel-instrumented app reads the trace ID from the active span instead.
func newTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}

func pushProfile(ctx context.Context, endpoint string) {
	var buf bytes.Buffer
	if err := pprof.StartCPUProfile(&buf); err != nil {
		log.Printf("start cpu profile: %v", err)
		return
	}
	select {
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
	}
	pprof.StopCPUProfile()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &buf)
	if err != nil {
		log.Printf("build profile push request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	// runtime/pprof already compresses its profile.proto output with gzip.
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("push profile: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("pushed profile: %s", resp.Status)
}

func main() {
	endpoint := os.Getenv("PPROF_PUSH_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://collector:4319/v1/pprof"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello %s\n", r.URL.Path)
	})
	mux.HandleFunc("/work", func(w http.ResponseWriter, r *http.Request) {
		traceID := newTraceID()
		// The trace_id label lands on every CPU sample taken while this
		// request runs, so a profile can be filtered down to one request.
		pprof.Do(r.Context(), pprof.Labels("trace_id", traceID), func(context.Context) {
			crunchNumbers()
		})
		fmt.Fprintf(w, "worked %s trace_id=%s\n", r.URL.Path, traceID)
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Capture back-to-back, with no idle gap. A 5-second window is always
	// open, so any burst of /work traffic always lands inside one.
	//
	// The old loop slept 10s between captures. A burst that fell in that
	// gap produced a profile with 0 samples, the collector forwarded it,
	// and nothing at all appeared in Pyroscope — with no error anywhere to
	// explain it. Continuous capture is also what "continuous profiling"
	// actually means.
	go func() {
		for {
			pushProfile(ctx, endpoint)
			if ctx.Err() != nil {
				return
			}
		}
	}()

	server := &http.Server{Addr: ":8006", Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	log.Println("demo6 listening on :8006")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
