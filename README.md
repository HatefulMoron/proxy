# Go Proxy Server

A lightweight HTTP proxy service that routes requests through third-party proxy services (like Oxylabs) with authentication.

## Quick Start

```bash
# Build
go build -o proxy-server main.go

# Run (uses config/config.json)
./proxy-server

# Run with custom port
PROXY_PORT=8082 ./proxy-server
```

## Configuration

Edit `config/config.json` or use environment variables:

```json
{
  "server": {
    "port": 8080,
    "host": "0.0.0.0"
  },
  "target": {
    "scheme": "https",
    "host": "solrenview.com"
  },
  "proxy": {
    "url": "http://dc.oxylabs.io:8000",
    "username": "user-ecosuite_p5OUw-country-US",
    "password": "xxx"
  },
  "logging": {
    "level": "info",
    "format": "json"
  }
}
```

### Environment Variables

- `PROXY_PORT` - Server port
- `PROXY_HOST` - Server host
- `TARGET_HOST` - Target domain
- `PROXY_URL` - Proxy service URL
- `PROXY_USERNAME` - Proxy authentication username
- `PROXY_PASSWORD` - Proxy authentication password
- `LOG_LEVEL` - Logging level (debug, info, warn, error)

## Usage

The server intercepts requests to your proxy domain and forwards them to the target domain through the configured proxy service.

**Request:** `http://localhost:8080/path?key=value`
**Forwards to:** `https://solrenview.com/path?key=value` (via Oxylabs proxy)

## Endpoints

- `GET /health` - Health check
- `GET /metrics` - Basic metrics
- `*` - Proxy all other requests

## Testing

```bash
go test -v
```
