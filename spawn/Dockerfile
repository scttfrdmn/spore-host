# Multi-stage build for spawn CLI tool
# Produces minimal container with spawn binary and AWS tools

FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy go mod files and i18n dependency
COPY spawn/go.mod spawn/go.sum spawn/
COPY pkg/i18n pkg/i18n/

# Download dependencies
WORKDIR /build/spawn
RUN go mod download

# Copy spawn source code
COPY spawn/ .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o spawn main.go

# Final stage - minimal runtime image
FROM alpine:latest

LABEL maintainer="Scott Friedman <scott@example.com>"
LABEL description="spawn - AWS EC2 instance launcher with parameter sweep support"
LABEL version="0.12.0"

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    aws-cli \
    openssh-client \
    bash \
    jq \
    curl

# Copy spawn binary from builder
COPY --from=builder /build/spawn/spawn /usr/local/bin/spawn

# Verify binary works
RUN spawn --version || spawn --help

# AWS credentials will be mounted at runtime
ENV AWS_SHARED_CREDENTIALS_FILE=/root/.aws/credentials
ENV AWS_CONFIG_FILE=/root/.aws/config

# Default to mycelium-dev profile for EC2 operations
ENV AWS_PROFILE=mycelium-dev

WORKDIR /workspace

ENTRYPOINT ["spawn"]
CMD ["--help"]
