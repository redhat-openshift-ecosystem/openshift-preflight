# Preflight Usage Examples

Below are detailed examples on how to leverage `preflight` in various
environments.

These examples are shown using the Operator policy against an operator bundle
(e.g. `preflight check operator <bundle`>), but the same concepts can be applied
to the Container policy (give or take a few configuration changes).

You will also need an index image containing your bundle for each of these approaches.
See [DOCS](BUILDING_AN_INDEX.md)

## As a Binary on Your Workstation

To run `preflight` on your workstation, you'll first need to download and
[install](../README.md#Installation)
the binary to your path.

You will also need:

- Your bundle image published to a container registry,
- An Index Image containing your operator's new bundle published to a container
  registry.
- A Kubeconfig for a user with cluster-admin privileges to an OpenShift 4.5+
  Cluster running Operator Lifecycle Manager.

The `preflight` tool will use the above to execute the Operator policy against
your test asset. With the above in hand, you can execute preflight against your
test asset by executing the following.

```bash
export KUBECONFIG=/path/to/your/kubeconfig 
export PFLT_INDEXIMAGE=registry.example.org/your-namespace/your-index-image:sometag
preflight check operator registry.example.org/your-namespace/your-bundle-image:sometag
```

## Using Podman (or Docker)

Running `preflight` in a Podman or Docker container is very similar to running
it on your workstation, but you will likely want to leverage a few volume mounts
to pass through various artifacts back to your host system for review at a later
time.

The below example has had its container-tool abstraced out, but you should be
able to run this with either Podman or Docker without issue.

Here, we explicitly set the location in the container where we would like
artifacts and logfiles to be written by using the `PFLT_ARTIFACTS` and
`PFLT_LOGFILE` environment variables. Then we bind host volumes to these
locations so that the data will be preserved when the container completes (the
container will be deleted after completion due to the `--rm` flag).

```bash
CONTAINER_TOOL=podman
$CONTAINER_TOOL run \
  -it \
  --rm \
  --env KUBECONFIG=/kubeconfig \
  --env PFLT_LOGLEVEL=trace \
  --env PFLT_INDEXIMAGE=registry.example.org/your-namespace/your-index-image:sometag \
  --env PFLT_ARTIFACTS=/artifacts \
  --env PFLT_CHANNEL=beta \
  --env PFLT_LOGFILE=/artifacts/preflight.log \
  -v /some/path/on/your/host/artifacts:/artifacts \
  -v /some/path/on/your/host/kubeconfig:/kubeconfig \
  quay.io/opdev/preflight:stable check operator registry.example.org/your-namespace/your-bundle-image:sometag
```

## As a Job In OpenShift (or Kubernetes)

You should be able to run `preflight` as a job in OpenShift without requiring
additional privileges or security context constraints.

As with previous examples, you will need to provide `preflight`a Kubeconfig
mapping a user with cluster-admin privileges to an OpenShift cluster that can be
used for tests.

In the namespace you will be creating this job, provision a secret with the
Kubeconfig:

```shell
oc create secret generic test-cluster-kubeconfig --from-file=kubeconfig=/some/path/on/your/host/kubeconfig
```

Then, create a Job manifest on your system. Note that you will need to
substitute:

- your bundle path in the `command` array
- your index image in the `env` array
- the volume type for the `outputdir` volume (if you want to use an alternate
  storage provider)

```yaml
echo >> preflight.yaml <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: preflight
spec:
  template:
    spec:
      containers:
      - name: preflight
        image: "quay.io/opdev/preflight:stable"
        command: ["preflight", "check", "operator", "registry.example.org/your-namespace/your-bundle-image:sometag"]
        env:
          - name: KUBECONFIG
            value: "/creds/kubeconfig"
          - name: PFLT_LOGLEVEL
            value: trace
          - name: PFLT_INDEXIMAGE
            value: "registry.example.org/your-namespace/your-index-image:sometag"
          - name: PFLT_LOGFILE
            value: "/artifacts/preflight.log"
          - name: PFLT_ARTIFACTS
            value: "/artifacts"
          - name: PFLT_CHANNEL
            value: "beta"
        volumeMounts:
          - name: "outputdir"
            mountPath: "/artifacts"
          - name: "kubeconfig"
            mountPath: "/creds"
      restartPolicy: Never
      volumes:
        - name: "outputdir"
          emptyDir:
            medium: ""
        - name: kubeconfig
          secret:
            secretName: test-cluster-kubeconfig
            optional: false
  backoffLimit: 2
EOF
```

```shell
oc apply -f preflight.yaml
```
