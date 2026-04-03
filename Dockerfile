FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bot ./cmd/bot
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /webhook ./cmd/webhook

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 1000 appuser

COPY --from=builder /bot /usr/local/bin/bot
COPY --from=builder /webhook /usr/local/bin/webhook

USER appuser

ENTRYPOINT ["bot"]
