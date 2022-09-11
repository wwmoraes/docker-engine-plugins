# Local test

```shell
curl -X POST \
  --unix-socket /run/docker/plugins/op.sock \
  -d '{ \
    "SecretName":"grafana", \
    "SecretLabels":{ \
      "connect.1password.io/vault":"Lab", \
      "connect.1password.io/item":"Grafana", \
      "connect.1password.io/field":"username" \
      } \
    }' \
  http://localhost/SecretProvider.GetSecret
```
