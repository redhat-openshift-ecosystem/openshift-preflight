ARG quay_expiration=never
ARG release_tag=0.0.0
ARG ARCH=amd64

# golang:1.16 image created 2021-06-24T00:31:06.02014601Z 
FROM docker.io/golang:1.17 AS builder
ARG quay_expiration
ARG release_tag
ARG ARCH

# Build the preflight binary
COPY . /go/src/preflight
WORKDIR /go/src/preflight
RUN make build RELEASE_TAG=${release_tag}

# ubi8:latest
FROM registry.access.redhat.com/ubi8/ubi:latest
ARG quay_expiration

# Define that tags should expire after 1 week. This should not apply to versioned releases.
LABEL quay.expires-after=${quay_expiration}

# Fetch the build image Architecture
LABEL ARCH=${ARCH}

# Define versions for dependencies
ARG OPENSHIFT_CLIENT_VERSION=4.7.19
ARG OPERATOR_SDK_VERSION=1.14.0

# Add preflight binary
COPY --from=builder /go/src/preflight/preflight /usr/local/bin/preflight

# Install dependencies
RUN dnf install -y \
      bzip2 \
      gzip \
      iptables \
      findutils \
    && dnf clean all

# Install OpenShift client binary
RUN curl -L https://mirror.openshift.com/pub/openshift-v4/clients/ocp/${OPENSHIFT_CLIENT_VERSION}/openshift-client-linux-${OPENSHIFT_CLIENT_VERSION}.tar.gz | tar -xzv -C /usr/local/bin oc

# Install Operator SDK binray
RUN curl -Lo /usr/local/bin/operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/v${OPERATOR_SDK_VERSION}/operator-sdk_linux_${ARCH} \
    && chmod 755 /usr/local/bin/operator-sdk

ENTRYPOINT ["/usr/local/bin/preflight"]
CMD ["--help"]
