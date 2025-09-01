# Teleport Auto-reviewer

A service that automatically reviews and rejects Teleport access requests based on configurable regular expression patterns.

## Features

- **Regex-based Rejection Rules**: Configure rejection rules using regular expressions for both request reasons and role names
- **Auto-refreshing Identity**: Automatically refreshes the service's identity file without requiring restart
- **Health Check Endpoint**: HTTP endpoint for monitoring service health and status
- **Configurable Rejection Messages**: Customize rejection messages per rule for specific feedback
- **Graceful Shutdown**: Handles SIGINT/SIGTERM signals for clean service shutdown
- **Structured Logging**: Comprehensive logging for monitoring and debugging

## Configuration

The service is configured via a YAML file. Here's an example configuration:

```yaml
# Configuration for the teleport-autoreviewer service

teleport:
  addr: "teleport.example.com:443"
  identity: "/var/lib/teleport/bot/identity"
  reviewer: "teleport-autoreviewer"
  identity_refresh_interval: "1h"

server:
  health_port: 8080
  health_path: "/health"

rejection:
  default_message: "Access request rejected due to policy violation"
  rules:
    - name: "Block production access without justification"
      reason_regex: "^$"
      message: "Production access requests must include detailed justification"
```

### Configuration Options

#### Teleport Section
- `addr`: Teleport cluster address
- `identity`: Path to the identity file for the service
- `reviewer`: Name used for the reviewer in rejection messages
- `identity_refresh_interval`: How often to refresh the identity file (e.g., "1h", "30m")

#### Server Section
- `health_port`: Port for the health check HTTP server (default: 8080)
- `health_path`: Path for the health check endpoint (default: "/health")

#### Rejection Section
- `default_message`: Default message used when a rule doesn't specify a custom message
- `rules`: Array of rejection rules

#### Rejection Rules
Each rule can have:
- `name`: Descriptive name for the rule
- `reason_regex`: Regular expression to match against request reasons
- `roles_regex`: Regular expression to match against requested roles
- `message`: Custom rejection message for this rule

**Note**: A request is rejected if it matches ANY rule. Rules can match either the reason OR the roles.

## Usage

### Building

```bash
go build -o teleport-autoreviewer
```

### Running

```bash
./teleport-autoreviewer
```

The service will:
1. Load configuration from `config/config.yaml`
2. Connect to Teleport using the configured identity
3. Start the health check HTTP server
4. Begin watching for access requests
5. Start the identity refresh routine

### Health Check

The service provides a health check endpoint at `http://localhost:8080/health` (configurable).

Example health check response:
```json
{
  "status": "healthy",
  "teleport_connected": true,
  "identity_valid": true,
  "last_request_processed": "2024-01-15T10:30:45Z",
  "last_identity_refresh": "2024-01-15T10:00:00Z",
  "uptime": "2h30m15s"
}
```

Health status meanings:
- `healthy`: Service is operational and connected to Teleport
- `unhealthy`: Service has issues (not connected to Teleport or invalid identity)

### Docker Usage

The service includes a production-ready multi-platform Dockerfile using distroless base images for minimal attack surface.

#### Building Docker Images

**Single Platform Build (local architecture):**
```bash
docker build -t teleport-autoreviewer:latest .
```

**Multi-Platform Build:**
```bash
# Build for Linux AMD64
docker build \
  --build-arg TARGETOS=linux \
  --build-arg TARGETARCH=amd64 \
  -t teleport-autoreviewer:latest-linux-amd64 .

# Build for Linux ARM64
docker build \
  --build-arg TARGETOS=linux \
  --build-arg TARGETARCH=arm64 \
  -t teleport-autoreviewer:latest-linux-arm64 .

# Build for multiple platforms using buildx
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t teleport-autoreviewer:latest \
  --push .
```

**Available Platform Arguments:**

| TARGETOS | TARGETARCH | Description            |
| -------- | ---------- | ---------------------- |
| linux    | amd64      | Linux x86_64 (default) |
| linux    | arm64      | Linux ARM64            |
| linux    | arm        | Linux ARM              |
| darwin   | amd64      | macOS Intel            |
| darwin   | arm64      | macOS Apple Silicon    |
| windows  | amd64      | Windows x86_64         |

#### Docker Features

- **Distroless Base**: Uses `gcr.io/distroless/static-debian12:nonroot` for minimal attack surface
- **Non-root User**: Runs as UID 65532 (nonroot user)
- **Static Binary**: CGO disabled for fully static compilation
- **Multi-stage Build**: Optimized for size and security
- **Health Check**: Built-in container health check
- **Security Context**: Read-only root filesystem compatible

#### Running with Docker

```bash
# Run with config and identity mounted
docker run -d \
  --name teleport-autoreviewer \
  -p 8080:8080 \
  -v /path/to/config.yaml:/app/config.yaml:ro \
  -v /path/to/identity:/etc/teleport/identity:ro \
  --read-only \
  --tmpfs /tmp \
  teleport-autoreviewer:latest

# Check health
curl http://localhost:8080/health
```

#### Docker Compose Example

```yaml
version: '3.8'
services:
  teleport-autoreviewer:
    image: teleport-autoreviewer:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config/config.yaml:/app/config.yaml:ro
      - ./identity:/etc/teleport/identity:ro
    read_only: true
    tmpfs:
      - /tmp
    restart: unless-stopped
    healthcheck:
      test: ["/usr/local/bin/teleport-autoreviewer", "-health-check"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
```

## Troubleshooting

### Common Issues

1. **Service won't start**: Check identity file path and permissions
2. **Not rejecting requests**: Verify regex patterns and check logs
3. **Health check fails**: Ensure port is available and not blocked by firewall
4. **Identity refresh failures**: Check file permissions and Teleport connectivity

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
