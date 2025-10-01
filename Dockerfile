# Build stage
FROM golang:1.25.0-alpine AS build
LABEL org.opencontainers.image.source="https://github.com/nrmnqdds/gomaluum" \
  org.opencontainers.image.description="Gomaluum API Server" \
  org.opencontainers.image.version="2.0" \
  org.opencontainers.image.licenses="Bantown Public License"

WORKDIR /app

# for sqlite purpose
RUN apk add --no-cache gcc musl-dev

# Copy and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -a -installsuffix cgo -o /app/gomaluum

FROM alpine:latest AS final
RUN apk add --update --no-cache ca-certificates curl

# Copy binary from build stage
COPY --from=build /app/gomaluum /

# Set environment variables
ENV APP_ENV=production
ENV PORT=1323
ENV HOSTNAME=0.0.0.0
ENV SSL_CERT_DIR=/etc/ssl/certs

# Expose ports
EXPOSE 50051
EXPOSE 1323

# Health check
HEALTHCHECK --interval=30s --timeout=3s \
  CMD curl -f http://localhost:1323/health || exit 1

# Entrypoint
ENTRYPOINT ["/gomaluum", "-p"]
