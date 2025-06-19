FROM --platform=$BUILDPLATFORM docker:28.2.2-dind-rootless

ARG GOVERSION=1.23.7
ARG TAGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

USER root

RUN apk add --no-cache curl delve gcc git musl-dev tar

ENV GOBUNDLE=go${GOVERSION}.${TARGETOS}-${TARGETARCH}.tar.gz
RUN rm -rf /usr/local/go/

RUN curl --fail --silent --show-error --location https://go.dev/dl/${GOBUNDLE} -o /tmp/${GOBUNDLE}
RUN tar -C /usr/local/ -xzf /tmp/${GOBUNDLE}
RUN rm /tmp/${GOBUNDLE}

ENV GOPATH=/go
ENV GOROOT=/usr/local/go
ENV CGO_ENABLED=1

ENV PATH=${GOROOT}/bin:${GOPATH}/bin:${PATH}

RUN mkdir -p /tmp/runtime-dir
ENV XDG_RUNTIME_DIR=/tmp/runtime-dir
ENV DOCKER_HOST=unix://${XDG_RUNTIME_DIR}/docker.sock

COPY dind/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

USER rootless
ENTRYPOINT ["/entrypoint.sh"]