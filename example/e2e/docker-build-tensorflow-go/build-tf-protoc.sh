set -x
set -e

source shared-vars.sh

mkdir -p $TENSORFLOW_GIT_REPO
git clone --depth 1 --branch v${VERSION} https://github.com/tensorflow/tensorflow.git $TENSORFLOW_GIT_REPO
pushd $TENSORFLOW_GIT_REPO

go mod init github.com/tensorflow/tensorflow
