# Build stage
FROM golang:1.23.2-alpine AS build
LABEL org.opencontainers.image.source="https://github.com/nrmnqdds/gomaluum" \
  org.opencontainers.image.description="Gomaluum API Server" \
  org.opencontainers.image.version="2.0" \
  org.opencontainers.image.licenses="Bantown Public License"

WORKDIR /app

# Copy and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .

# Build main binary
RUN CGO_ENABLED=0 GOOS=linux go build --ldflags="-checklinkname=0" -o /app/gomaluum

# Build healthcheck binary (points to /health/main.go)
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/health ./health

# Certificate stage
FROM alpine:latest AS certs
RUN apk --update add ca-certificates

# Final stage
FROM gcr.io/distroless/static-debian11 AS final

# Copy binary from build stage
COPY --from=build /app/gomaluum /
COPY --from=build /app/health /

# Copy certificates from the certs stage
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Set environment variables
ENV APP_ENV=production
ENV PORT=1323
ENV HOSTNAME=0.0.0.0
ENV SSL_CERT_DIR=/etc/ssl/certs

# Run as non-root user
USER nonroot:nonroot

# Expose ports
EXPOSE 50051
EXPOSE 1323

# Health check (use custom Go health binary)
HEALTHCHECK --interval=30s --timeout=3s CMD ["/health"]

# Entrypoint
ENTRYPOINT ["/gomaluum", "-p"]
