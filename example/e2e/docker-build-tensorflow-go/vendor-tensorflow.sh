VERSION=${VERSION:-2.4.2}
go mod edit -require github.com/tensorflow/tensorflow@v${VERSION}+incompatible
go mod vendor
