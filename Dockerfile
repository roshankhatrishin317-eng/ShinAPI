FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for go mod download (some deps may need it)
RUN apk add --no-cache git

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.BuildDate=${BUILD_DATE}'" -o ./CLIProxyAPI ./cmd/server/

FROM alpine:3.22.0

RUN apk add --no-cache tzdata ca-certificates

RUN mkdir -p /CLIProxyAPI /root/.cli-proxy-api

COPY --from=builder /app/CLIProxyAPI /CLIProxyAPI/CLIProxyAPI

# Create default config for cloud deployment
RUN echo 'host: ""' > /CLIProxyAPI/config.yaml && \
    echo 'port: 8317' >> /CLIProxyAPI/config.yaml && \
    echo 'redis:' >> /CLIProxyAPI/config.yaml && \
    echo '  enabled: false' >> /CLIProxyAPI/config.yaml && \
    echo 'metrics-db:' >> /CLIProxyAPI/config.yaml && \
    echo '  enabled: false' >> /CLIProxyAPI/config.yaml && \
    echo 'debug: false' >> /CLIProxyAPI/config.yaml && \
    echo 'auth-dir: "/root/.cli-proxy-api"' >> /CLIProxyAPI/config.yaml && \
    echo 'api-keys:' >> /CLIProxyAPI/config.yaml && \
    echo '  - "default-key"' >> /CLIProxyAPI/config.yaml && \
    echo 'remote-management:' >> /CLIProxyAPI/config.yaml && \
    echo '  allow-remote: true' >> /CLIProxyAPI/config.yaml && \
    echo '  secret-key: "otsu317"' >> /CLIProxyAPI/config.yaml

WORKDIR /CLIProxyAPI

# Expose main port
EXPOSE 8317

ENV TZ=UTC

CMD ["./CLIProxyAPI"]