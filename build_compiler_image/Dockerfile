FROM ghcr.io/oracle/oraclelinux8-instantclient:23

ENV GOLANG_VERSION=1.24.5
ENV GO_INSTALL_DIR=/usr/local

RUN set -eux; \
    yum update -y && \
    yum install -y gcc && \
    yum clean all

COPY ./bin/go${GOLANG_VERSION}.linux-amd64.tar.gz /tmp/go${GOLANG_VERSION}.linux-amd64.tar.gz

RUN set -eux; \
    rm -rf ${GO_INSTALL_DIR}/go; \
    mkdir -p ${GO_INSTALL_DIR}; \
    tar -C ${GO_INSTALL_DIR} -xzf /tmp/go${GOLANG_VERSION}.linux-amd64.tar.gz; \
    rm /tmp/go${GOLANG_VERSION}.linux-amd64.tar.gz; \
    ${GO_INSTALL_DIR}/go/bin/go version

ENV PATH="${GO_INSTALL_DIR}/go/bin:${PATH}"
