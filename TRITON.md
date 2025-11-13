
# Triton gRPC

The Triton integration uses protocol buffers for gRPC communication.
Generated proto files are **committed to the repository** for build reliability.

# When to Regenerate Proto Files

Regenerate only when:
- Modifying `proto/triton/grpc_service.proto`
- Upgrading to a new Triton API version
- Upgrading protobuf/gRPC to a new major version

# Prerequisites

Install `protoc` (Protocol Buffer Compiler):

```bash
# macOS
brew install protobuf

# Linux (Debian/Ubuntu)
apt-get install -y protobuf-compiler

# Verify installation
protoc --version  # Should be 3.x or higher
```

Install Go protoc plugins:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Ensure GOPATH/bin is in PATH
export PATH="$PATH:$(go env GOPATH)/bin"

# Verify installation
which protoc-gen-go
which protoc-gen-go-grpc
```

# Regeneration Steps

```bash
# 1. Navigate to repo root
cd /path/to/viant/mly

# 2. Delete old generated files (ensures clean regeneration)
rm -f proto/triton/grpc_service.pb.go
rm -f proto/triton/grpc_service_grpc.pb.go

# 3. Regenerate
protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  proto/triton/grpc_service.proto

# 4. Verify generation succeeded
ls -lh proto/triton/*.pb.go
```

# Verification

After regeneration:

```bash
# Ensure code compiles
go build ./...

# Run tests
go test ./service/platform/...

# Review changes
git diff proto/triton/
```

# Troubleshooting

**Error: `protoc: command not found`**
- Install protoc using package manager (see Prerequisites)

**Error: `protoc-gen-go: program not found`**
- Ensure `$GOPATH/bin` is in your `$PATH`
- Run: `export PATH="$PATH:$(go env GOPATH)/bin"`

**Error: `Import "..." was not found`**
- Run protoc from the repository root directory
- Verify proto file imports are correct

**Notes:**
- Generated files are ~30KB and should be committed
- Proto definitions are based on [Triton's official protocol](https://github.com/triton-inference-server/common/blob/main/protobuf/grpc_service.proto)
- Triton uses `raw_output_contents` for performance (binary format vs. structured)