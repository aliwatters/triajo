FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ginla-mcp .

FROM alpine:3.20

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/ginla-mcp .

ENV MONGO_URI=mongodb://saturn.local:27017
ENV MONGO_DATABASE=ginla

ENTRYPOINT ["./ginla-mcp"]
