set -x
set -e

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
source $script_dir/goenv_cgo.lib.sh

LIBTENSORFLOW_VERSION=${LIBTENSORFLOW_VERSION:-2.4.2}

RESOLVED_GOPATH=$(go env GOPATH)

TENSORFLOW_GIT_REPO=$RESOLVED_GOPATH/src/github.com/tensorflow/tensorflow
mkdir -p $TENSORFLOW_GIT_REPO
git clone --depth 1 --branch v${LIBTENSORFLOW_VERSION} https://github.com/tensorflow/tensorflow.git $TENSORFLOW_GIT_REPO || true
pushd $TENSORFLOW_GIT_REPO

if [ ! -f go.mod ]; then
    go mod init github.com/tensorflow/tensorflow
fi

go get google.golang.org/protobuf/reflect/protoreflect@v1.26.0
go mod tidy || true

export LD_LIBRARY_PATH=/usr/local/lib

goenv_cgo::push /usr/local/lib /opt/homebrew/lib

go generate -x ./tensorflow/go/op || true
cp -r ./tensorflow/go/vendor/github.com/tensorflow/tensorflow/tensorflow/go/* ./tensorflow/go
go generate -x ./tensorflow/go/op

go test ./tensorflow/go

goenv_cgo::pop

rm -rf .git

mkdir -p $RESOLVED_GOPATH/pkg/mod/github.com/tensorflow/tensorflow@v${LIBTENSORFLOW_VERSION}+incompatible
cp -r ./ $RESOLVED_GOPATH/pkg/mod/github.com/tensorflow/tensorflow@v${LIBTENSORFLOW_VERSION}+incompatible

popd