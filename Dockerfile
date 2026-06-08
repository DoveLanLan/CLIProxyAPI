# syntax=docker/dockerfile:1.7

FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./

ARG GOPROXY="https://goproxy.cn|direct"
ARG GOSUMDB="sum.golang.google.cn"
ENV GOPROXY="${GOPROXY}"
ENV GOSUMDB="${GOSUMDB}"

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    for attempt in 1 2 3 4 5 6 7 8 9 10; do \
        go mod download && break; \
        if [ "$attempt" = "10" ]; then exit 1; fi; \
        sleep $((attempt * 2)); \
    done

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    if [ -z "$BUILD_DATE" ]; then BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ); fi && \
    CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.BuildDate=${BUILD_DATE}'" -o ./CLIProxyAPI ./cmd/server/

FROM alpine:3.22.0

RUN apk add --no-cache tzdata

RUN mkdir /CLIProxyAPI

COPY --from=builder ./app/CLIProxyAPI /CLIProxyAPI/CLIProxyAPI

COPY config.example.yaml /CLIProxyAPI/config.example.yaml

WORKDIR /CLIProxyAPI

EXPOSE 8317

ENV TZ=Asia/Shanghai

RUN cp /usr/share/zoneinfo/${TZ} /etc/localtime && echo "${TZ}" > /etc/timezone

CMD ["./CLIProxyAPI"]
