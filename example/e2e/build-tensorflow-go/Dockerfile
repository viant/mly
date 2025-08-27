ARG GO_VERSION=1.17
FROM golang:${GO_VERSION}-bullseye
RUN apt-get update && apt-get -y install --no-install-recommends libprotobuf-dev protobuf-compiler

COPY install-dependencies.sh /opt/bin/install-dependencies.sh
RUN /bin/bash /opt/bin/install-dependencies.sh

COPY with-cgo.sh /opt/bin/with-cgo.sh
COPY build-tf-protoc.sh /opt/bin/build-tf-protoc.sh
RUN /bin/bash /opt/bin/build-tf-protoc.sh

COPY vendor-tensorflow.sh /opt/bin/vendor-tensorflow.sh
WORKDIR /opt/src
CMD ["/bin/bash", "/opt/bin/vendor-tensorflow.sh"]