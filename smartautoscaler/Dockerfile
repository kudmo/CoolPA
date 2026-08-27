FROM golang:1.26.6-bookworm AS builder

RUN apt-get update && apt-get install -y \
    libopenblas-dev \
    libprotobuf-dev \
    protobuf-compiler \
    wget \
    && rm -rf /var/lib/apt/lists/*

ENV ONNX_VERSION=1.29.0
RUN wget -q https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/onnxruntime-linux-x64-${ONNX_VERSION}.tgz \
    && tar -xzf onnxruntime-linux-x64-${ONNX_VERSION}.tgz \
    && cp -r onnxruntime-linux-x64-${ONNX_VERSION}/include/* /usr/local/include/ \
    && cp -r onnxruntime-linux-x64-${ONNX_VERSION}/lib/* /usr/local/lib/ \
    && ln -sf /usr/local/lib/libonnxruntime.so.${ONNX_VERSION} /usr/local/lib/libonnxruntime.so \
    && ldconfig \
    && rm onnxruntime-linux-x64-${ONNX_VERSION}.tgz \
    && rm -rf onnxruntime-linux-x64-${ONNX_VERSION}

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -ldflags="-s -w" \
    -o smartautoscaler \
    ./cmd/main.go

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    libopenblas0-pthread \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/local/lib/libonnxruntime.so* /usr/local/lib/
COPY --from=builder /app/latency_model.onnx /app/
COPY --from=builder /app/smartautoscaler /app/

RUN ldconfig

WORKDIR /app

RUN useradd -u 10001 -m appuser && chown -R appuser:appuser /app
USER appuser

ENTRYPOINT ["/app/smartautoscaler"]
