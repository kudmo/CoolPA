FROM golang:1.21 AS builder

WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o smartautoscaler ./cmd/main.go

FROM gcr.io/distroless/base-debian11
WORKDIR /app
COPY --from=builder /app/smartautoscaler /app/
# COPY config/example-config.yaml /app/config.yaml
# ENV CONFIG_PATH=/app/config.yaml
ENTRYPOINT ["/app/smartautoscaler"]
