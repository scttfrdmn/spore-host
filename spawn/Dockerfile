# Multi-stage build for spawn and truffle CLI tools
# Produces minimal container with spawn, truffle binaries and AWS tools

FROM golang:1.26-alpine AS builder

# Use Docker's automatic platform ARGs for multi-platform builds
ARG TARGETARCH

WORKDIR /build

# Copy shared i18n dependency
COPY pkg/i18n pkg/i18n/

# Build spawn
COPY spawn/go.mod spawn/go.sum spawn/
WORKDIR /build/spawn
RUN go mod download
COPY spawn/ .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o spawn main.go

# Build truffle
WORKDIR /build
COPY truffle/go.mod truffle/go.sum truffle/
WORKDIR /build/truffle
RUN go mod download
COPY truffle/ .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o truffle main.go

# Final stage - minimal runtime image
FROM alpine:latest

LABEL maintainer="Scott Friedman <scott@example.com>"
LABEL description="spawn and truffle - AWS EC2 management and discovery tools"
LABEL version="0.12.0"

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    aws-cli \
    openssh-client \
    bash \
    jq \
    curl

# Copy binaries from builder
COPY --from=builder /build/spawn/spawn /usr/local/bin/spawn
COPY --from=builder /build/truffle/truffle /usr/local/bin/truffle

# Verify binaries work
RUN spawn --version || spawn --help
RUN truffle --version || truffle --help

# AWS credentials will be mounted at runtime
ENV AWS_SHARED_CREDENTIALS_FILE=/root/.aws/credentials
ENV AWS_CONFIG_FILE=/root/.aws/config

# Default to mycelium-dev profile for EC2 operations
ENV AWS_PROFILE=mycelium-dev

WORKDIR /workspace

ENTRYPOINT ["spawn"]
CMD ["--help"]
