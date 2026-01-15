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

# Copy config
COPY config.yaml /CLIProxyAPI/config.yaml

WORKDIR /CLIProxyAPI

# Expose main port
EXPOSE 8317

ENV TZ=UTC

CMD ["./CLIProxyAPI"]