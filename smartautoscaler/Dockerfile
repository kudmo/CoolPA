FROM golang:1.21-bullseye AS builder

RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    libopenblas-dev \
    libprotobuf-dev \
    protobuf-compiler \
    wget \
    && rm -rf /var/lib/apt/lists/*

RUN wget https://github.com/microsoft/onnxruntime/releases/download/v1.24.3/onnxruntime-linux-x64-1.24.3.tgz \
    && tar -xzf onnxruntime-linux-x64-1.24.3.tgz \
    && cp -r onnxruntime-linux-x64-1.24.3/include/* /usr/local/include/ \
    && cp -r onnxruntime-linux-x64-1.24.3/lib/* /usr/local/lib/ \
    && ln -sf /usr/local/lib/libonnxruntime.so.1.24.3 /usr/local/lib/libonnxruntime.so \
    && ldconfig

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 \
    go build \
    -ldflags="-r /usr/local/lib" \
    -o smartautoscaler \
    ./cmd/main.go

FROM debian:bullseye-slim

RUN apt-get update && apt-get install -y \
    libopenblas-base \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/local/lib/libonnxruntime.so* /usr/local/lib/
COPY --from=builder /app/random_forest_model.onnx /app/
COPY --from=builder /app/smartautoscaler /app/

RUN ln -sf /usr/local/lib/libonnxruntime.so.1.24.3 /usr/local/lib/libonnxruntime.so \
    && ldconfig

WORKDIR /app

RUN useradd -u 10001 -m appuser && chown -R appuser:appuser /app
USER appuser

ENTRYPOINT ["/app/smartautoscaler"]
