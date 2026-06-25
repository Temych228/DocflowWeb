FROM golang:1.26.2-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY . .

WORKDIR /app/services/notification-service
RUN go mod download && go mod tidy

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags='all=-lang=go1.26' -o notification-service ./cmd/app

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/services/notification-service/notification-service .

EXPOSE 8086 9006 9206

CMD ["./notification-service"]