# Build stage
FROM golang:1.25.0-alpine AS build
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
RUN CGO_ENABLED=0 GOOS=linux go build --ldflags="-checklinkname=0" -o /app/gomaluum

# Certificate stage
FROM alpine:latest AS certs
RUN apk --update add ca-certificates curl tzdata

# Final stage
FROM gcr.io/distroless/static-debian11 AS final

# Copy binary from build stage
COPY --from=build /app/gomaluum /

# Copy certificates, curl, and timezone data from the certs stage
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=certs /usr/bin/curl /usr/bin/curl
COPY --from=certs /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=certs /etc/localtime /etc/localtime

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

# Health check
HEALTHCHECK --interval=5m --timeout=5s \
  CMD curl -f http://localhost:1323/health || exit 1

# Entrypoint
ENTRYPOINT ["/gomaluum", "-p"]
