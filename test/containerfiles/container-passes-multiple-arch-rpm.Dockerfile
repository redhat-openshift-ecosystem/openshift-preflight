FROM registry.access.redhat.com/ubi8/ubi:8.7-1112

RUN useradd preflightuser

COPY --chown=preflightuser:preflightuser example-license.txt /licenses/

LABEL name="preflight test image container-policy" \
      vendor="preflight test vendor" \
      version="1" \
      release="1" \
      summary="testing the preflight tool" \
      description="test the preflight tool"

RUN dnf update glibc -y && dnf install glibc.i686 -y

USER preflightuser

