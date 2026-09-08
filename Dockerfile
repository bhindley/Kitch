FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

# Build the binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o kitch .

# --- Runtime stage ---
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/kitch .

EXPOSE 8080

ENTRYPOINT ["./kitch"]
