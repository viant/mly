# End-to-End Testing

Powered by [endly](https://github.com/viant/endly).

# Warning

After running endly:

1. The `vendor` directory will be populated (i.e. via `go mod vendor`).
2. Packages will be installed.
3. Your `go` version will be set to 1.22.
4. If running on Linux, a Docker image will be tagged.
5. If running on Linux, `/usr/local` will have `libtensorflow` files populated.

**Note: The binary built for MacOS is NOT meant to be portable.**

# Prerequisites

1. [Install endly](https://github.com/viant/endly/tree/master/doc/installation)
2. Install OS (and CPU Architecture) dependencies.
3. Run endly from this directory.

Linux ARM CPUs not supported.

## MacOS

1. Install [Homebrew](https://brew.sh/).

## Linux

1. Install packages providing `sudo`, `curl`.
2. Install [Docker](https://docs.docker.com/engine/install/).
