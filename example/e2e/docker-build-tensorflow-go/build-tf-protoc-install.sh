set -x
set -e

source shared-vars.sh
pushd $TENSORFLOW_GIT_REPO

# add debugging due to Go support sometimes being ignored
sed -i '18 i set -x' tensorflow/go/genop/generate.sh

if [ "$VERSION" \< "2.8.0" ] && ([ "$VERSION" \> "2.7.0" ] || [ "$VERSION" == "2.7.0" ]); then
	cat /opt/bin/patch-2.7.4.patch | git apply 
fi

go get google.golang.org/protobuf/reflect/protoreflect@v1.26.0 
go get github.com/golang/protobuf/ptypes

export LD_LIBRARY_PATH=/usr/local/lib 

go generate -v -x ./tensorflow/go/op || true
#go mod vendor -e || true
cp -rv ./tensorflow/go/vendor/github.com/tensorflow/tensorflow/tensorflow/go/* ./tensorflow/go
go generate -v -x ./tensorflow/go/op 

go test ./tensorflow/go
rm -rfv .git

mkdir -p $RESOLVED_GOPATH/pkg/mod/github.com/tensorflow/tensorflow@v${VERSION}+incompatible
cp -rv ./ $RESOLVED_GOPATH/pkg/mod/github.com/tensorflow/tensorflow@v${VERSION}+incompatible

