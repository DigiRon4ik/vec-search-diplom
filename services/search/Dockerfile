FROM golang:1.23-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential cmake git wget ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /tmp/usearch
RUN git clone --depth 1 https://github.com/unum-cloud/usearch.git . && \
    cmake -B build \
      -DCMAKE_BUILD_TYPE=Release \
      -DUSEARCH_BUILD_LIB_C=ON \
      -DUSEARCH_BUILD_TEST_CPP=OFF \
      -DUSEARCH_BUILD_BENCH_CPP=OFF && \
    cmake --build build --config Release --target usearch_c -j$(nproc) && \
    cp build/libusearch_c.so /usr/local/lib/ && \
    cp c/usearch.h /usr/local/include/ && \
    ldconfig

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .

ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-I/usr/local/include"
ENV CGO_LDFLAGS="-L/usr/local/lib -lusearch_c"
RUN go build -o /search ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates wget libgomp1 && rm -rf /var/lib/apt/lists/*
COPY --from=builder /usr/local/lib/libusearch_c.so /usr/local/lib/
RUN ldconfig

COPY --from=builder /search /usr/local/bin/search

EXPOSE 8083
CMD ["search"]
