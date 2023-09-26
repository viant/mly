ARG GO_VERSION=1.17

FROM golang:${GO_VERSION}-bullseye
RUN apt-get update && apt-get -y install --no-install-recommends libprotobuf-dev protobuf-compiler 

ARG TF_VERSION=2.4.3
ENV VERSION=${TF_VERSION}

RUN /bin/bash -c "echo ${VERSION} ${TF_VERSION}"

COPY install-tensorflowlib.sh /opt/bin/install-tensorflowlib.sh

WORKDIR /opt/bin
RUN /bin/bash install-tensorflowlib.sh

COPY shared-vars.sh /opt/bin/shared-vars.sh

COPY build-tf-protoc.sh /opt/bin/build-tf-protoc.sh
RUN /bin/bash build-tf-protoc.sh

COPY build-tf-protoc-install.sh /opt/bin/build-tf-protoc-install.sh
COPY patch-2.7.4-go-protos.patch /opt/bin/patch-2.7.4.patch
RUN /bin/bash build-tf-protoc-install.sh

COPY vendor-tensorflow.sh /opt/bin/vendor-tensorflow.sh
WORKDIR /opt/src
CMD ["/bin/bash", "/opt/bin/vendor-tensorflow.sh"]
