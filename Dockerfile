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

# ubi10:latest
FROM registry.access.redhat.com/ubi10/ubi:latest
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

# Define versions for dependencies
ARG OPERATOR_SDK_VERSION=1.42.3

# Install and verify Operator SDK binary
# Verification follows https://sdk.operatorframework.io/docs/installation/#2-verify-the-downloaded-binary
ARG OPERATOR_SDK_GPG_KEY=3B2F1481D146238080B346BB052996E2A20B5C7E
RUN dnf install -y gnupg2 && dnf clean all \
    && export OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/download/v${OPERATOR_SDK_VERSION} \
    && curl --fail -Lo /usr/local/bin/operator-sdk ${OPERATOR_SDK_DL_URL}/operator-sdk_linux_${ARCH} \
    && curl --fail -Lo /tmp/checksums.txt ${OPERATOR_SDK_DL_URL}/checksums.txt \
    && curl --fail -Lo /tmp/checksums.txt.asc ${OPERATOR_SDK_DL_URL}/checksums.txt.asc \
    && gpg --keyserver keyserver.ubuntu.com --recv-keys ${OPERATOR_SDK_GPG_KEY} \
    && gpg -u "Operator SDK (release) <cncf-operator-sdk@cncf.io>" --verify /tmp/checksums.txt.asc \
    && grep "operator-sdk_linux_${ARCH}" /tmp/checksums.txt | sed "s|operator-sdk_linux_${ARCH}|/usr/local/bin/operator-sdk|" | sha256sum -c - \
    && chmod 755 /usr/local/bin/operator-sdk \
    && rm -rf /tmp/checksums.txt /tmp/checksums.txt.asc "$HOME/.gnupg" \
    && dnf remove -y gnupg2 && dnf clean all

# Add preflight binary
COPY --from=builder /go/src/preflight/preflight /usr/local/bin/preflight

#copy license
COPY LICENSE /licenses/LICENSE

ENTRYPOINT ["/usr/local/bin/preflight"]
CMD ["--help"]
