FROM scratch

COPY example-license.txt /licenses/

LABEL name="preflight test image scratch plus root container-policy" \
      vendor="preflight test vendor" \
      version="1" \
      release="1" \
      summary="testing the preflight tool" \
      description="test the preflight tool"

USER root
