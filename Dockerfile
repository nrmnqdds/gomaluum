FROM golang:1.23.2-alpine AS build
LABEL org.opencontainers.image.source=https://github.com/nrmnqdds/gomaluum

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download
RUN go mod verify

COPY . .

# install templ version that go.mod is using
RUN go install github.com/a-h/templ/cmd/templ@$(go list -m -f '{{ .Version }}' github.com/a-h/templ)
RUN templ generate

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/gomaluum

# use debug so can docker exec
FROM gcr.io/distroless/static-debian11:debug AS final
COPY --from=build /app/gomaluum /
COPY --from=build /app/static /static

ENV APP_ENV=production
ENV PORT=1323
ENV HOSTNAME=0.0.0.0

USER nonroot:nonroot

EXPOSE 1323

ENTRYPOINT ["/gomaluum", "-p"]

FROM node:20-alpine AS base

ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
RUN corepack enable

FROM base AS deps
# Check https://github.com/nodejs/docker-node/tree/b4117f9333da4138b03a546ec926ef50a31506c3#nodealpine to understand why libc6-compat might be needed.
RUN apk add --no-cache libc6-compat
WORKDIR /frontend

# Install dependencies
COPY frontend/package.json frontend/pnpm-lock.yaml* ./
RUN pnpm i --frozen-lockfile

# Rebuild the source code only when needed
FROM base AS builder
WORKDIR /frontend
COPY --from=deps /frontend/node_modules ./node_modules
COPY frontend/. .

ENV NODE_ENV=production
ENV TZ=Asia/Kuala_Lumpur
ENV DEBIAN_FRONTEND=noninteractive
RUN pnpm run build

FROM nginx:alpine AS runner
COPY --from=builder /frontend/dist /usr/share/nginx/html
EXPOSE 80
