# Multi-stage build — produces node, simplekvs, and client binaries
FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
RUN go build -o /bin/node      ./cmd/node
RUN go build -o /bin/simplekvs ./cmd/simplekvs
RUN go build -o /bin/client    ./cmd/client

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /bin/node      /usr/local/bin/node
COPY --from=builder /bin/simplekvs /usr/local/bin/simplekvs
COPY --from=builder /bin/client    /usr/local/bin/client
COPY infrastructure/experiments/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
