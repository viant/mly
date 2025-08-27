#!/bin/bash

set -x
set -e

LIBTENSORFLOW_VERSION=${LIBTENSORFLOW_VERSION:-2.4.2}

go mod edit -require github.com/tensorflow/tensorflow@v${LIBTENSORFLOW_VERSION}+incompatible
go mod vendor