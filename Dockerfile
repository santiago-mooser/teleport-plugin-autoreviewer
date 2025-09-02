# syntax=docker/dockerfile:1
### Build
ARG GO_VERSION=1.23.10
ARG ALPINE_VERSION=3.22

# Build stage
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

# Dependency
RUN apk add --no-cache git gcc g++ make

# Install dependencies
WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy source code
COPY . .
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 make build

# Final stage - distroless
FROM gcr.io/distroless/static-debian12:nonroot

# Set working directory
WORKDIR /app

# Copy the binary
COPY --from=builder \
    --chown=nonroot:nonroot \
    /src/teleport-plugin-request-autoreviewer \
    /app/teleport-plugin-request-autoreviewer

# Set entrypoint
ENTRYPOINT ["/app/teleport-plugin-request-autoreviewer"]
