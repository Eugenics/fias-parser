# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS builder
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/gar_reader .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/gar_reader /usr/local/bin/fias-parser
COPY deployments/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
COPY deployments/run-scheduled.sh /usr/local/bin/run-scheduled.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh /usr/local/bin/run-scheduled.sh && \
    mkdir -p /app/configs /app/source/xml/full /app/source/xml/delta /archives

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["crond", "-f", "-l", "2"]
