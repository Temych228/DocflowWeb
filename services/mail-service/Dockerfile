FROM golang:1.26.2-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY . /app

WORKDIR /app/services/mail-service
RUN go mod download && go mod tidy

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags='all=-lang=go1.26' -o mail-service ./cmd/app

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/services/mail-service/mail-service .

EXPOSE 8087 9007 9207

CMD ["./mail-service"]