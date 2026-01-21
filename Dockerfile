FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /wallets .

# Final image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates curl

COPY --from=builder /wallets /usr/bin/wallets

ENV OTEL_SERVICE_NAME=wallets

ENTRYPOINT ["/usr/bin/wallets"]
CMD ["serve"]
