#!/bin/bash

set -x
set -e

IS_MACOS=$(uname -s | grep -i 'darwin')

appPath=${appPath:-$(pwd)}
goVersion=${goVersion:-1.22}

export LIBTENSORFLOW_VERSION=2.4.2

if [ -n "$IS_MACOS" ]; then
    brew install protobuf
    brew install libtensorflow

    bash ${appPath}/example/e2e/build-tensorflow-go/build-tf-protoc.sh

    (
        cd ${appPath}
        bash ${appPath}/example/e2e/build-tensorflow-go/vendor-tensorflow.sh
    )
else
    # download and install libtensorflow
    curl https://storage.googleapis.com/tensorflow/libtensorflow/libtensorflow-cpu-linux-x86_64-2.4.2.tar.gz | sudo tar -xmzv -C /usr/local

    # build tensorflow-go module in docker
    docker build -t mly-docker-build-tensorflow-go:1.0 \
      --build-arg GO_VERSION=${goVersion} \
      -f ${appPath}/example/e2e/build-tensorflow-go/Dockerfile \
      ${appPath}/example/e2e/build-tensorflow-go

    # run go mod vendor in the container to pull correct tensorflow module
    docker run --rm -v ${appPath}:/opt/src mly-docker-build-tensorflow-go:1.0
fi
