Build the binary with the repo commit hash

```bash
go build -ldflags "-X github.com/komish/preflight/version.commit=$(git rev-parse HEAD)" cmd/preflight.go
```
