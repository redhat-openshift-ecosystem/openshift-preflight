# Building an Index Image

Preflight's Operator policy (i.e. `preflight check operator ...`) requires
access to an index image containing the Operator bundle under test.

The fastest way to do this is to utilize the `opm` tool to build an index image,
publish that image, and then provide that to Preflight when executing your
checks. To install the `opm` cli, visit the
[operator-framework/operator-registry
releases](https://github.com/operator-framework/operator-registry/releases) page
and download an appropriate binary for your system.

## Building an Index With Your Bundle

For reference, the below instructions were executed using `opm` at the following
version:

```shell
$ opm version
Version: version.Version{OpmVersion:"v1.18.0", GitCommit:"b826849", BuildDate:"2021-08-11T19:07:57Z", GoOs:"darwin", GoArch:"amd64"}
```

Your bundle must already be published to a registry in order to build your index
image with that bundle.

Run the following command to create an index with the specified bundle,
substituting your bundle's registry path, and the desired registry path of your
index. 

(NOTE: Docker users may need to specify the `--container-tool=docker` flag to
this command)

```shell
opm index add --bundles registry.example.org/your-namespace/your-bundle:0.0.1 --tag registry.example.org/your-namespace/your-index:0.0.1
```

(sample output)

```shell
INFO[0000] building the index                            bundles="[registry.example.org/your-namespace/your-bundle:0.0.1]"
... truncated
INFO[0002] [podman build -f index.Dockerfile359827617 -t registry.example.org/your-namespace/your-index:0.0.1 .]  bundles="[registry.example.org/your-namespace/your-bundle:0.0.1]"
```

Then push this bundle to your registry of choice. This registry must be
accessible to Preflight as well as the target cluster.

```shell
podman push registry.example.org/your-namespace/your-index:0.0.1
```
If the index image is stored in a private repository, set the value of `PFLT_DOCKERCONFIG` to the path of the docker configuration file.

Finally, set the value of `PFLT_INDEXIMAGE` to this value and run preflight:

```shell
export PFLT_INDEXIMAGE=registry.example.org/your-namespace/your-index:0.0.1
preflight check operator registry.example.org/your-namespace/your-bundle:0.0.1
```

For detailed information on how to use the `opm` tool, see [Building an Index of
Operators using
opm](https://github.com/operator-framework/operator-registry#building-an-index-of-operators-using-opm)
