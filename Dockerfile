# golang:1.16 image created 2021-06-24T00:31:06.02014601Z 
FROM docker.io/library/golang@sha256:be99fa59acd78bb22a41bbc1e15ebfab2262498ee0c2e28c3d09bc44d51d1774 AS builder

# Build the preflight binary
COPY . /go/src/preflight
WORKDIR /go/src/preflight
RUN make build

# podman:v3.2.2
FROM quay.io/podman/stable@sha256:c281b604477e10e3e84f81af9b68db59ee616bf6a34391224e5e5c74db0428d0

# Define versions for dependencies
ARG OPENSCAP_VERSION=1.3.5
ARG OPENSHIFT_CLIENT_VERSION=4.7.19
ARG OPERATOR_SDK_VERSION=1.9.0

# Add preflight binary
COPY --from=builder /go/src/preflight/preflight /usr/local/bin/preflight

# Install dependencies
RUN dnf install -y \
      bzip2 \
      gzip \
      iptables \
      findutils \
      openscap-scanner \
      podman \
      buildah \
      skopeo \
    && dnf clean all

# Install oscap-podman binary
RUN curl -L https://github.com/OpenSCAP/openscap/releases/download/${OPENSCAP_VERSION}/openscap-${OPENSCAP_VERSION}.tar.gz | tar -xzv -C /usr/local/bin openscap-${OPENSCAP_VERSION}/utils/oscap-podman \
    && mv /usr/local/bin/openscap-${OPENSCAP_VERSION}/utils/oscap-podman /usr/local/bin/oscap-podman

# Install OpenShift client binary
RUN curl -L https://mirror.openshift.com/pub/openshift-v4/clients/ocp/${OPENSHIFT_CLIENT_VERSION}/openshift-client-linux-${OPENSHIFT_CLIENT_VERSION}.tar.gz | tar -xzv -C /usr/local/bin oc

# Install Operator SDK binray
RUN curl -Lo /usr/local/bin/operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/v${OPERATOR_SDK_VERSION}/operator-sdk_linux_amd64 \
    && chmod 755 /usr/local/bin/operator-sdk

ENTRYPOINT ["/usr/local/bin/preflight"]
CMD ["--help"]
