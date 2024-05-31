FROM registry.access.redhat.com/ubi8-minimal@sha256:9e458f41ff8868ceae00608a6fff35b45fd8bbe967bf8655e5ab08da5964f4d0

# This container file backs
# quay.io/opdev/preflight-test-fixture:duplicate-layers, and is intended to test
# an edge case with HasModifiedFiles where multiple duplicate layers were geting
# squashed into a single entry in our layer-to-file map, causing invalid
# modification flags.
#
# The produced artifact is about 100mb, so this fixture exists just to avoid
# having that blob stored in-repo.

COPY example-license.txt /LICENSE
RUN microdnf install gzip -y
COPY example-license.txt /LICENSE
RUN microdnf install gzip -y
