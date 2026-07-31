
# Triton gRPC

The Triton integration uses protocol buffers for gRPC communication.
Generated proto files are **committed to the repository** for build reliability.

Triton uses `raw_output_contents` for performance (binary format vs. structured).

# Output Tensor Ordering

The KServe/Triton v2 protocol only guarantees that `raw_output_contents[i]`
aligns with `outputs[i]` in a `ModelInfer` response. It does **not** guarantee
that the `outputs` order matches the order reported by `ModelMetadata` — a model
whose metadata lists `[a, b]` may return inference outputs as `[b, a]`. Every
response tensor carries its own `name`, so mly addresses outputs by name: the
client returns outputs keyed by tensor name and `TritonEvaluator.Predict`
reorders them into signature order (see the `service/triton` package doc). Do
not reintroduce positional handling of response tensors — when two outputs share
a datatype, a positional mismatch mislabels values with no error.

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

Initialize submodule in `third_party/triton-common`.

```bash
git submodule update --init --recursive
```

Deleted old generated files, which ensures clean regeneration.

```bash
rm -f proto/triton/grpc_service.pb.go
rm -f proto/triton/grpc_service_grpc.pb.go
```

Regenerate the files.

```bash
protoc \
  -I "$PWD/third_party/triton-common/protobuf" \
  --go_out=paths=source_relative,Mgrpc_service.proto=github.com/viant/mly/proto/triton,Mmodel_config.proto=github.com/viant/mly/proto/triton:"$PWD/proto/triton" \
  --go-grpc_out=paths=source_relative,Mgrpc_service.proto=github.com/viant/mly/proto/triton,Mmodel_config.proto=github.com/viant/mly/proto/triton:"$PWD/proto/triton" \
  "$PWD/third_party/triton-common/protobuf/model_config.proto" \
  "$PWD/third_party/triton-common/protobuf/grpc_service.proto"
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
