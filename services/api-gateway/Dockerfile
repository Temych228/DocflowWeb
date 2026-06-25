# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-gateway ./cmd/main.go

# Stage 2: Runtime
FROM alpine:3.20

RUN apk --no-cache add ca-certificates

WORKDIR /root

# Copy binary from builder
COPY --from=builder /build/api-gateway .
COPY --from=builder /build/web ./web

# Copy .env if exists
COPY .env* ./

EXPOSE 8080

CMD ["./api-gateway"]
