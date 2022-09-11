# cifs-volume-plugin

> CIFS filesystem volume provider for Docker

[![Docker Image Size (latest semver)](https://img.shields.io/docker/image-size/wwmoraes/cifs-volume-plugin)](https://hub.docker.com/r/wwmoraes/cifs-volume-plugin)
[![Docker Image Version (latest semver)](https://img.shields.io/docker/v/wwmoraes/cifs-volume-plugin?label=image%20version)](https://hub.docker.com/r/wwmoraes/cifs-volume-plugin)
[![Docker Pulls](https://img.shields.io/docker/pulls/wwmoraes/cifs-volume-plugin)](https://hub.docker.com/r/wwmoraes/cifs-volume-plugin)

## About

Mounts CIFS-compatible file shares as container volumes.

## Getting Started

Install the plugin:

```shell
docker plugin install wwmoraes/cifs-volume-plugin:$(uname -m)-latest \
  --alias cifs \
  --grant-all-permissions
```

By default `/run/secrets/cifs` is the source of credential files. You can set
`credentials.source` to override it.

Set `DEFAULT_OPTIONS` if you need to apply flags to mounts without options.

### Credential files

The driver will use credential files based on the UNC path set on the volume.
All credential files must start with the host name without the double forward
slashes, and may optionally have the share path. If no credential file is found
for a specific share, the driver will fallback to the parent shares up to a
host-only credential file.

Use `%2F` as path separator, as filenames cannot contain forward slashes (`/`).

For instance, considering this environment:

```sh
$ ls -1 /run/secrets/cifs
some-host
some-host%2Ffoo

$ cat /run/secrets/cifs/some-host
username=admin
password=secret-admin-password

$ cat /run/secrets/cifs/some-host%2Ffoo
username=foo
password=secret-foo-password
```

And the following mounts:

```sh
docker volume create -d cifs -o share=//some-host/foo foo
# will be mounted using the foo user

docker volume create -d cifs -o share=//some-host/bar bar
# will be mounted using the admin user

docker volume create -d cifs -o share=//another-host qux
# will be mounted anonymously
```

Those create commands will all succeed, as creating a volume only stores its
metadata. If the credentials are missing or incorrect, then mounting the volume
will fail.

### Prerequisites

- Docker Engine with volume plugin support (tested on v20)
- `mount` with `cifs` type support

## Usage

Create volumes using the `--driver` option:

```shell
docker volume create -d cifs -o share=//file-server/foo foo
```

Or the equivalent on the mechanism you're using, such as compose.
