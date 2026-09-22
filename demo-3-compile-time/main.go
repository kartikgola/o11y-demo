// A deliberately ordinary Go HTTP server.
//
// There is no OpenTelemetry import here, and no OTel dependency in
// go.mod. That is the whole point: this one file is the input to two
// different zero-code approaches.
//
//	Demo 3  builds it with `otelc go build`   (Dockerfile)
//	Demo 4  builds it with a plain `go build` (Dockerfile.plain), then
//	        attaches to the running process from outside, with eBPF.
//
// Same source, same handler, two ways to instrument it.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8003"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello %s\n", r.URL.Path)
	})

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
