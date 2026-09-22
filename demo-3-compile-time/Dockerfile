# Demo 3: compile-time instrumentation.
#
# The only difference from an ordinary Go build is the last line: it runs
# `otelc go build` instead of `go build`. Compare this file with
# Dockerfile.plain, which builds the very same main.go for Demo 4.
FROM golang:1.26-alpine AS build
WORKDIR /src

# Pinned, not @latest. A new otelc release between the rehearsal and the
# talk would change this build under you.
RUN go install go.opentelemetry.io/otelc/tool/cmd/otelc@v1.1.0

COPY go.mod ./
RUN go mod download || true
COPY main.go ./
RUN CGO_ENABLED=0 otelc go build -o /app .

FROM alpine:3.20
COPY --from=build /app /app
EXPOSE 8003
CMD ["/app"]
