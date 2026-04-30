# TensorFlow

Directory `go-tensorflow`.

The official TensorFlow team `libtensorflow` no longer provides up to date bindings for CGo.

We keep the 2.4.2 bindings, as their graph, operations, and kernel API has not changed since then.

# Construction Notes

How this was "vendored" into the `third_party` directory:

1. Use `scripts/build-tf-protoc.sh` to generate Go files from protocol buffer files.
    1. Use `brew` to install `tensorflow` and `protobuf`.
    1. Get `tensorflow/tensorflow` v2.4.2 from Github.
    2. Setup protocol buffer compiler requirements.
    3. Mock pushing to Go mod cache
2. Copy from Go mod cache to `third_party/go-tensorflow`.
3. Strip out files unneeded for Go
    1. Go to `third_party/go-tensorflow`
    2. Remove files except `tensorflow` (`rm -rv $(ls . | grep -v tensorflow)`).
    3. Remove dot files manually.
    4. Go to `tensorflow`
    5. Remove files except under `go` (`rm -rv $(ls . | grep -v '^go$')`).
4. Add `replace` in `viant/mly`'s `go.mod`.
