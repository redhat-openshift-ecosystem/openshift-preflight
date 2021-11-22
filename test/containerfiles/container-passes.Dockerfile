FROM registry.access.redhat.com/ubi8/ubi:latest

RUN useradd preflightuser

COPY --chown=preflightuser:preflightuser example-license.txt /licenses/

LABEL name="preflight test image" \ 
      vendor="preflight test vendor" \
      version="1" \
      release="1" \
      summary="testing the preflight tool" \
      description="test the preflight tool"

USER preflightuser
