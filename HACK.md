Build the binary with the repo commit hash

```bash
go build -ldflags "-X github.com/redhat-openshift-ecosystem/openshift-preflight/version.commit=$(git rev-parse HEAD)" main.go
```
