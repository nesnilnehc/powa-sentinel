# Stage 1: Build
FROM golang:1.22-alpine AS builder

# Install git for version info
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with version info
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" \
    -o powa-sentinel ./cmd/powa-sentinel

# Stage 2: Runtime
FROM gcr.io/distroless/static:nonroot

# Copy binary from builder
COPY --from=builder /app/powa-sentinel /powa-sentinel

# Copy example config as default
COPY config/config.yaml.example /config.yaml

# Use non-root user
USER nonroot:nonroot

# Expose health check port
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/powa-sentinel"]
CMD ["-config", "/config.yaml"]
