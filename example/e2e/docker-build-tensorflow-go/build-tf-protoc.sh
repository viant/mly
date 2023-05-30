set -x
set -e

VERSION=${VERSION:-2.4.2}

RESOLVED_GOPATH=$(go env GOPATH)

TENSORFLOW_GIT_REPO=$RESOLVED_GOPATH/src/github.com/tensorflow/tensorflow
mkdir -p $TENSORFLOW_GIT_REPO
git clone --depth 1 --branch v${VERSION} https://github.com/tensorflow/tensorflow.git $TENSORFLOW_GIT_REPO
pushd $TENSORFLOW_GIT_REPO

go mod init github.com/tensorflow/tensorflow
go get google.golang.org/protobuf/reflect/protoreflect@v1.26.0 

export LD_LIBRARY_PATH=/usr/local/lib 
go generate -v -x ./tensorflow/go/op || true
cp -rv ./tensorflow/go/vendor/github.com/tensorflow/tensorflow/tensorflow/go/* ./tensorflow/go
go generate -v -x ./tensorflow/go/op 
go test ./tensorflow/go
rm -rfv .git

mkdir -p $RESOLVED_GOPATH/pkg/mod/github.com/tensorflow/tensorflow@v${VERSION}+incompatible
cp -rv ./ $RESOLVED_GOPATH/pkg/mod/github.com/tensorflow/tensorflow@v${VERSION}+incompatible

