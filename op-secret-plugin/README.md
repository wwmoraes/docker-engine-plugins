# op-secret-plugin

> 1Password Connect secrets provider for Docker

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

Note: The secret must exist on 1Password before creation. The plugin is
read-only, so both the file and stdin are ignored.
