FROM scratch
# Labels for testing.
LABEL operators.operatorframework.io.test.mediatype.v1=scorecard+v1
LABEL operators.operatorframework.io.test.config.v1=tests/scorecard/

# Copy files to locations specified by labels.
COPY failed-bundle-assets/manifests /manifests/
COPY failed-bundle-assets/metadata /metadata/
COPY failed-bundle-assets/tests/scorecard /tests/scorecard/
