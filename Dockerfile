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
