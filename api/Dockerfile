FROM golang:1.25.1-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY api ./api
COPY cmd ./cmd
COPY kv ./kv
COPY storage ./storage

RUN go build -o /simplekvs ./cmd/simplekvs

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /simplekvs /usr/local/bin/simplekvs

RUN mkdir -p /data

ENV KVS_ADDR=:8080
ENV KVS_WAL_ENABLED=true
ENV KVS_WAL_PATH=/data/wal.log
ENV KVS_SNAPSHOT_ENABLED=true
ENV KVS_SNAPSHOT_PATH=/data/snapshot.json

EXPOSE 8080

CMD ["simplekvs"]
