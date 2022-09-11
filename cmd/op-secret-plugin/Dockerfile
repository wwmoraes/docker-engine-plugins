# syntax = docker/dockerfile:experimental
ARG ALPINE_VERSION=3.16
ARG GO_VERSION=1.18


FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS dev

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY internal internal
COPY cmd/op-secret-plugin op-secret-plugin

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}
RUN --mount=type=cache,target=/root/.cache/go-build go build \
  -mod=mod \
  -o op-secret-plugin ./op-secret-plugin/...


FROM alpine:${ALPINE_VERSION}

RUN apk update && apk add --quiet --no-cache \
  tini=0.19.0-r0 \
  && rm -rf /var/cache/apk/* /var/lib/apk/*

COPY --from=dev /src/op-secret-plugin/op-secret-plugin /usr/local/bin/

ENTRYPOINT ["tini", "--", "op-secret-plugin"]
