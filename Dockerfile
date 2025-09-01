# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.23-alpine AS builder

# Install git and ca-certificates (needed for go modules and HTTPS)
RUN apk add --no-cache git ca-certificates tzdata

# Create appuser for the final image
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations for static linking
# Use build arguments to support multi-platform builds
ARG TARGETOS=linux
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o teleport-autoreviewer .

# Final stage - distroless
FROM gcr.io/distroless/static-debian12:nonroot

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy ca-certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /build/teleport-autoreviewer /usr/local/bin/teleport-autoreviewer

# Use nonroot user (uid 65532)
USER nonroot:nonroot

# Set working directory
WORKDIR /app

# Expose health check port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/usr/local/bin/teleport-autoreviewer", "-health-check"] || exit 1

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/teleport-autoreviewer"]
