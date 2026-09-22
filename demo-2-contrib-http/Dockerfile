FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 go build -o /app main.go

FROM alpine:3.20
COPY --from=build /app /app
EXPOSE 8002
CMD ["/app"]
