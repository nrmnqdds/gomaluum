FROM golang:1.23.2-alpine AS build-go
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/gomaluum

FROM node:20-alpine AS build-node
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
RUN corepack enable
WORKDIR /frontend
COPY frontend/package.json frontend/pnpm-lock.yaml* ./
RUN apk add --no-cache libc6-compat
RUN pnpm i --frozen-lockfile
COPY frontend/. .
RUN pnpm run build

FROM alpine:latest
RUN apk add --no-cache nginx supervisor

# Copy the Go binary
COPY --from=build-go /app/gomaluum /usr/local/bin/

# Copy the frontend build
COPY --from=build-node /frontend/dist /usr/share/nginx/html

# Create supervisor config
RUN mkdir -p /etc/supervisor.d/
COPY supervisord.conf /etc/supervisord.conf

# Copy nginx config
COPY frontend/nginx.conf /etc/nginx/nginx.conf

ENV APP_ENV=production
ENV PORT=1323
ENV HOSTNAME=0.0.0.0

EXPOSE 80 1323

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
