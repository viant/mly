#!/bin/bash

set -x
set -e

IS_MACOS=$(uname -s | grep -i 'darwin')

appPath=${appPath:-$(pwd)}
goVersion=${goVersion:-1.22}

export LIBTENSORFLOW_VERSION=2.4.2

if [ -n "$IS_MACOS" ]; then
    brew install libtensorflow
else
    # download and install libtensorflow
    curl https://storage.googleapis.com/tensorflow/libtensorflow/libtensorflow-cpu-linux-x86_64-2.4.2.tar.gz | sudo tar -xmzv -C /usr/local
fi
