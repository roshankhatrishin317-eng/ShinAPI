# ShinAPI

<div align="center">

**High-Performance Unified AI Gateway for CLI Tools & Developers**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8.svg)](https://golang.org/)
[![Docker Pulls](https://img.shields.io/docker/pulls/eceasy/cli-proxy-api.svg)](https://hub.docker.com/r/eceasy/cli-proxy-api)

[English](README.md) | [中文](README_CN.md)

</div>

---

**ShinAPI (CLI Proxy API)** is a production-ready, unified proxy server that bridges multiple AI providers through a single standardized API. It provides automatic protocol translation between **OpenAI**, **Gemini**, **Claude**, **Codex**, **DeepSeek**, **Qwen**, and **iFlow** formats, enabling seamless integration for CLI tools, AI agents, and developer applications.

## Features

### Multi-Provider Integration
Unified access to leading AI models through a single API:
- **OpenAI** - GPT-4, GPT-4o, and Codex models
- **Google Gemini** - Gemini Pro, Flash, Ultra via AI Studio or Vertex AI
- **Anthropic Claude** - Claude 3.5/4 family with API Key and OAuth support
- **DeepSeek** - DeepSeek-V3 and reasoning models
- **Qwen** - Alibaba's Qwen models
- **iFlow** - Enterprise model routing
- **Codex/Antigravity** - Specialized coding models

### Protocol Translation
Automatic bidirectional conversion between all provider formats:
- OpenAI <-> Claude <-> Gemini <-> Codex
- Request/response normalization across protocols
- Streaming SSE translation

### Intelligent Load Balancing
- **Round-Robin** - Distribute traffic across multiple keys
- **Fill-First** - Maximize single key usage before rotation
- **Automatic Failover** - Retry on backup providers
- **Rate Limit Handling** - Smart quota management with cooldown

### Multi-Layer Caching
- **L1 LRU** - In-memory cache with sub-ms response
- **L2 Redis** - Distributed cache for multi-instance deployments
- **L3 Semantic** - Similarity-based response matching
- **L4 Streaming** - SSE event replay for repeated requests

### Authentication
- **OAuth 2.0** - Built-in flows for Gemini, Claude, Codex, Qwen, iFlow
- **API Key Rotation** - Automatic key management with cooldown
- **Token Refresh** - Automatic OAuth token renewal
- **Circuit Breaker** - Disable failing providers automatically

### Web Dashboard
Next.js 16 control panel with:
- Real-time analytics (TPS, latency, error rates)
- Provider health monitoring
- Configuration management
- API Playground for testing
- Audit log viewer

### Observability
- Prometheus metrics export
- WebSocket real-time metrics stream
- PostgreSQL historical metrics
- Request/response audit logging

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Clients                                  │
│   CLI Tools    │    VS Code/Cursor    │    AI Agents            │
└───────────────────────────┬─────────────────────────────────────┘
                            │ HTTP/WebSocket
┌───────────────────────────▼─────────────────────────────────────┐
│                      ShinAPI Gateway                             │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────────────┐ │
│  │Load Balancer│──│ Auth Manager │──│ Protocol Translator     │ │
│  └─────────────┘  └──────────────┘  └─────────────────────────┘ │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────────────┐ │
│  │Cache Layers │  │Model Router  │  │ Metrics & Observability │ │
│  └─────────────┘  └──────────────┘  └─────────────────────────┘ │
└───────────────────────────┬─────────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────────┐
│                        Providers                                 │
│  OpenAI  │  Gemini  │  Claude  │  DeepSeek  │  Qwen  │  iFlow   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Quick Start

### Installation

**Build from source:**
```bash
git clone https://github.com/router-for-me/CLIProxyAPI.git
cd CLIProxyAPI
go build -o cli-proxy-api ./cmd/server
```

**Using Docker:**
```bash
docker run -d -p 8317:8317 \
  -v $(pwd)/config.yaml:/CLIProxyAPI/config.yaml \
  -v $(pwd)/auths:/root/.cli-proxy-api \
  eceasy/cli-proxy-api:latest
```

**Using Docker Compose:**
```bash
docker-compose up -d
```

### Configuration

Create `config.yaml`:

```yaml
port: 8317
host: "0.0.0.0"
auth-dir: "~/.cli-proxy-api"
debug: false

# Client API keys (for authenticating incoming requests)
api-keys:
  - "sk-your-client-key"

# Provider configurations
gemini-api-key:
  - api-key: "AIzaSy..."
    models:
      - name: "gemini-2.0-flash"
        alias: "flash"
      - name: "gemini-2.5-pro"
        alias: "pro"

claude-api-key:
  - api-key: "sk-ant-..."
    models:
      - name: "claude-sonnet-4-20250514"
        alias: "sonnet"

openai-compatibility:
  - base-url: "https://api.openai.com/v1"
    api-key: "sk-..."
    models:
      - name: "gpt-4o"
      - name: "gpt-4o-mini"

# Optional: Redis cache
redis:
  address: "localhost:6379"
  password: ""
  db: 0

# Optional: Metrics database
metrics-db:
  dsn: "postgresql://user:pass@localhost:5432/metrics"
```

### Run

```bash
./cli-proxy-api --config config.yaml
```

Server starts at `http://localhost:8317`
Dashboard at `http://localhost:8317/dashboard`

### OAuth Login Commands

```bash
./cli-proxy-api --login              # Gemini OAuth
./cli-proxy-api --claude-login       # Claude OAuth
./cli-proxy-api --codex-login        # Codex OAuth
./cli-proxy-api --qwen-login         # Qwen OAuth
./cli-proxy-api --iflow-login        # iFlow OAuth
./cli-proxy-api --antigravity-login  # Antigravity OAuth
./cli-proxy-api --vertex-import <credentials.json>  # Import Vertex AI
```

---

## API Reference

### OpenAI-Compatible Endpoints

```
POST /v1/chat/completions      # Chat completions (streaming supported)
POST /v1/completions           # Text completions
GET  /v1/models                # List available models
POST /v1/responses             # OpenAI Responses API
```

### Claude-Compatible Endpoints

```
POST /v1/messages              # Claude Messages API
POST /v1/messages/count_tokens # Token counting
```

### Gemini-Compatible Endpoints

```
GET  /v1beta/models            # List Gemini models
POST /v1beta/models/*action    # Gemini generation
```

### Management API

```
# Configuration
GET/PUT /v0/management/api-keys
GET/PUT /v0/management/debug
GET/PUT /v0/management/port

# Provider Keys
GET/PUT/DELETE /v0/management/gemini-api-key
GET/PUT/DELETE /v0/management/claude-api-key
GET/PUT/DELETE /v0/management/openai-compatibility

# OAuth
GET  /v0/management/anthropic-auth-url
GET  /v0/management/gemini-cli-auth-url
POST /v0/management/oauth-callback
GET  /v0/management/get-auth-status

# Metrics
GET /v0/management/live-metrics
GET /v0/management/historical-metrics
GET /v0/management/usage-stats
GET /v0/management/provider-health

# Audit
GET    /v0/management/audit/logs
GET    /v0/management/audit/stats
DELETE /v0/management/audit/logs

# Playground
POST /v0/management/playground/execute
GET  /v0/management/playground/models
```

### WebSocket

```
GET /ws/metrics    # Real-time metrics stream
```

---

## Project Structure

```
shinapi/
├── cmd/server/main.go       # Application entry point
├── internal/
│   ├── api/                 # HTTP server and routes
│   ├── auth/                # Provider OAuth implementations
│   ├── cache/               # Multi-layer caching
│   ├── config/              # Configuration management
│   ├── translator/          # Protocol translation engines
│   ├── usage/               # Metrics and usage tracking
│   ├── store/               # Storage backends (Postgres/Git/S3)
│   └── observability/       # Prometheus metrics
├── sdk/                     # Embeddable Go SDK
│   ├── cliproxy/            # Core proxy service
│   ├── api/handlers/        # Request handlers
│   ├── auth/                # Token management
│   └── translator/          # Translation interfaces
├── dashboard/               # Next.js web dashboard
├── examples/                # Usage examples
├── docs/                    # Documentation
└── config.yaml              # Configuration file
```

---

## SDK Usage

Embed ShinAPI directly in your Go application:

```go
package main

import (
    "context"
    "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy"
)

func main() {
    svc, err := cliproxy.NewBuilder().
        WithConfigPath("config.yaml").
        Build()
    if err != nil {
        panic(err)
    }

    ctx := context.Background()
    svc.Run(ctx)
}
```

See [SDK Documentation](docs/sdk-usage.md) for advanced usage.

---

## Environment Variables

```bash
# Management API
MANAGEMENT_PASSWORD=<password>

# PostgreSQL Token Store
PGSTORE_DSN=postgresql://user:pass@host:5432/db
PGSTORE_SCHEMA=public

# Git-Backed Config Store
GITSTORE_GIT_URL=https://github.com/org/repo.git
GITSTORE_GIT_TOKEN=ghp_token

# S3-Compatible Object Store
OBJECTSTORE_ENDPOINT=https://s3.example.com
OBJECTSTORE_BUCKET=cli-proxy-config
OBJECTSTORE_ACCESS_KEY=<key>
OBJECTSTORE_SECRET_KEY=<secret>

# Deployment Mode
DEPLOY=cloud
```

---

## Docker Compose

```yaml
services:
  shinapi:
    image: eceasy/cli-proxy-api:latest
    ports:
      - "8317:8317"
    volumes:
      - ./config.yaml:/CLIProxyAPI/config.yaml
      - ./auths:/root/.cli-proxy-api
      - ./logs:/CLIProxyAPI/logs
    restart: unless-stopped
    environment:
      - TZ=UTC
```

---

## Documentation

- [Architecture Overview](docs/ARCHITECTURE.md)
- [API Reference](docs/API.md)
- [SDK Usage Guide](docs/sdk-usage.md)
- [SDK Advanced Features](docs/sdk-advanced.md)

---

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/new-feature`)
3. Commit changes
4. Push to branch
5. Open a Pull Request

---

## License

MIT License - see [LICENSE](LICENSE) for details.

---

<div align="center">
  <sub>Built by the ShinAPI Team</sub>
</div>
