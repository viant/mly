set -x -e

SUPPORT=cpu
VERSION=${VERSION:-2.4.2}
DESTINATION=/usr/local

mkdir -p $DESTINATION

curl -s -L https://storage.googleapis.com/tensorflow/libtensorflow/libtensorflow-${SUPPORT}-$(uname | tr '[:upper:]' '[:lower:]')-$(uname -m)-${VERSION}.tar.gz | tar xz --directory $DESTINATION

