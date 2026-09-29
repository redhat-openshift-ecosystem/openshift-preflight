ARG quay_expiration=never
ARG release_tag=0.0.0
ARG ARCH=amd64
ARG OS=linux

FROM registry.access.redhat.com/ubi10/go-toolset:1.26 AS builder
ARG quay_expiration
ARG release_tag
ARG ARCH
ARG OS

# Switching to root user, since default users is 1001,
# which prohibits copying from `/tmp` during make build cmd
USER root

# Override UBI10 Go toolset microarchitecture defaults to maintain backward
# compatibility with the same hardware baseline as UBI9-built releases.
# See https://github.com/redhat-openshift-ecosystem/openshift-preflight/issues/1339
ENV GOAMD64=v2
ENV GOPPC64=power8

# Build the preflight binary
COPY . /go/src/preflight
WORKDIR /go/src/preflight
RUN make build RELEASE_TAG=${release_tag}

# Assemble runtime packages outside UBI Micro because it does not include a
# package manager.
FROM registry.access.redhat.com/ubi10/ubi-micro:latest AS runtime-base

FROM registry.access.redhat.com/ubi10/ubi:latest AS runtime-packages

# CA certificates enable HTTPS access to registries and Pyxis.
# tar and gzip package Preflight artifacts for OpenShift CI.
RUN --mount=type=bind,from=runtime-base,target=/mnt/micro,rw \
  dnf install -y \
      --installroot=/mnt/rootfs \
      --releasever=10 \
      --setopt=install_weak_deps=False \
      --nodocs \
      ca-certificates \
      gzip \
      tar && \
    dnf clean all --installroot=/mnt/rootfs && \
    rm -rf /mnt/rootfs/var/cache/dnf /mnt/rootfs/var/cache/yum && \
    mkdir -p /mnt/rootfs && \
    cp -a /mnt/micro/. /mnt/rootfs/

FROM runtime-base
ARG quay_expiration
ARG release_tag
ARG preflight_commit
ARG ARCH
ARG OS

# Metadata
LABEL name="Preflight" \
      vendor="Red Hat, Inc." \
      maintainer="Red Hat OpenShift Ecosystem" \
      version="1" \
      summary="Provides the OpenShift Preflight certification tool." \
      description="Preflight runs certification checks against containers and Operators." \
      url="https://github.com/redhat-openshift-ecosystem/openshift-preflight" \
      release=${release_tag} \
      vcs-ref=${preflight_commit}


# Define that tags should expire after 1 week. This should not apply to versioned releases.
LABEL quay.expires-after=${quay_expiration}

# Fetch the build image Architecture
LABEL ARCH=${ARCH}
LABEL OS=${OS}

# Add the runtime packages installed for UBI Micro.
COPY --from=runtime-packages /mnt/rootfs/ /

# Add preflight binary
COPY --from=builder /go/src/preflight/preflight /usr/local/bin/preflight

#copy license
COPY LICENSE /licenses/LICENSE

ENTRYPOINT ["/usr/local/bin/preflight"]
CMD ["--help"]
