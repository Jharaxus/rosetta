# ── Build stage ────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /migrate ./cmd/migrate

# ── Production image ───────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12

COPY --from=builder /server  /server
COPY --from=builder /migrate /migrate

EXPOSE 8090

ENTRYPOINT ["/server"]
