# op-secret-plugin

> 1Password Connect secrets provider for Docker

[![Docker Image Size (latest semver)](https://img.shields.io/docker/image-size/wwmoraes/op-secret-plugin)](https://hub.docker.com/r/wwmoraes/op-secret-plugin)
[![Docker Image Version (latest semver)](https://img.shields.io/docker/v/wwmoraes/op-secret-plugin?label=image%20version)](https://hub.docker.com/r/wwmoraes/op-secret-plugin)
[![Docker Pulls](https://img.shields.io/docker/pulls/wwmoraes/op-secret-plugin)](https://hub.docker.com/r/wwmoraes/op-secret-plugin)

## About

Connects to an 1Password Connect instance to fetch and provide secrets to
containers.

## Getting Started

Install the plugin:

```shell
docker plugin install wwmoraes/op-secret-plugin:$(uname -m)-latest \
  --alias op \
  --grant-all-permissions \
    OP_CONNECT_HOST=https://op-connect-host:8080 \
    OP_CONNECT_TOKEN_FILE=/run/secrets/op/token
```

If you need to use a plaintext token, use `OP_CONNECT_TOKEN_FILE` instead of
`OP_CONNECT_TOKEN`.

### Prerequisites

- Docker Engine with secret plugin support (tested on v20)
- 1Password Connect service instance running and accessible through HTTP

## Usage

Create secrets using the `--driver` option:

```shell
docker secret create -d op \
  -l connect.1password.io/vault=bar \
  -l connect.1password.io/item=baz \
  -l connect.1password.io/field=qux \
  -l connect.1password.io/reusable=false \ # optional, defaults to true
  foo
```

Note: Creation works if the secret doesn't exist on 1Password. It'll be checked
on each first mount, and will fail the service if missing.
