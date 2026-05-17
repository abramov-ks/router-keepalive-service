FROM golang:1.22-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o server .

# ── Runtime ────────────────────────────────────────────────────────────────────
FROM alpine:3.20

WORKDIR /app
COPY --from=builder /build/server .

ENV PORT=8080
ENV DB_PATH=/data/keepalive.db

EXPOSE 8080

ENTRYPOINT ["./server"]
