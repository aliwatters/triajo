# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a statically linked binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o ginla-api ./...

# ---- runtime stage ----
FROM scratch

# Copy CA certificates so TLS works (e.g. Atlas)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /app/ginla-api /ginla-api

EXPOSE 8080

ENV PORT=8080 \
    MONGO_URI=mongodb://saturn.local:27017 \
    MONGO_DATABASE=ginla

ENTRYPOINT ["/ginla-api"]
