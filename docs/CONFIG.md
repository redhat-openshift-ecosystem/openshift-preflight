# Configuring Preflight

The following configurables are available for the `preflight` tool.

## Common Configuration

|Variable|Kind|Doc|Required or Optional|Default|
|--|--|--|--|--|
|`PFLT_LOGLEVEL`|env|The verbosity of the preflight tool itself. Ex. warn, debug, trace, info, error|optional|[warn](https://github.com/redhat-openshift-ecosystem/openshift-preflight/blob/main/cmd/defaults.go#L6)|
|`PFLT_LOGFILE`|env|Where the execution logfile will be written.|optional|[preflight.log](https://github.com/redhat-openshift-ecosystem/openshift-preflight/blob/main/cmd/defaults.go#L5)|
|`PFLT_ARTIFACTS`|env|Where check-specific artifacts will be written.|optional|[artifacts/](https://github.com/redhat-openshift-ecosystem/openshift-preflight/blob/main/cmd/defaults.go#L7)|


## Operator Policy Configuration

These configurables are specific to cases where `preflight check operator ...`
is called.

|Variable|Kind|Doc|Required or Optional|Default|
|--|--|--|--|--|
|`KUBECONFIG`|env|The operator policy must interact with a Kubernetes cluster for checks such as `DeployableByOLM` and running [OperatorSDK Scorecard](https://sdk.operatorframework.io/docs/testing-operators/scorecard/).|required|-|
|`PFLT_NAMESPACE`|env|The namespace to use when running [OperatorSDK Scorecard](https://sdk.operatorframework.io/docs/testing-operators/scorecard/)|optional|[default](https://github.com/redhat-openshift-ecosystem/openshift-preflight/blob/main/cmd/defaults.go#L8)|
|`PFLT_SERVICEACCOUNT`|env|The service account to use when running [OperatorSDK Scorecard](https://sdk.operatorframework.io/docs/testing-operators/scorecard/)|optional|[default](https://github.com/redhat-openshift-ecosystem/openshift-preflight/blob/main/cmd/defaults.go#L9)|
|`PFLT_INDEXIMAGE`|env|The index image to use when testing that an operator is `DeployableByOLM`|required|-|
|`PFLT_DOCKERCONFIG`|env|The full path to a dockerconfigjson file, which is pushed to the target test cluster to access images in private repositories in the `DeployableByOLM`. If empty, no secret is created and the resource is assumed to be public.|optional|-|
|`PFLT_SCORECARD_IMAGE`|env|A uri that points to the scorecard image digest, used in disconnected environments. It should only be used in a disconnected environment. Use `preflight runtime-assets` on a connected workstation to generate the digest that needs to be mirrored.|optional|-|
|`PFLT_SCORECARD_WAIT_TIME`|env|A time value that will be passed to scorecard's `--wait-time` environment variable.|optional|[default](https://github.com/redhat-openshift-ecosystem/openshift-preflight/blob/main/cmd/defaults.go#L10)|
|`PFLT_CHANNEL`|env|The name of the operator channel which is used by `DeployableByOLM` to deploy the operator. If empty, the default operator channel in bundle's annotations file is used.|optional|-|


For information on how to build an index image, see [BUILDING_AN_INDEX.md](BUILDING_AN_INDEX.md).

## Container Policy Configuration

These configurables are specific to cases where `preflight check container ...`
is called.

|Variable|Kind|Doc|Required or Optional|Default|
|--|--|--|--|--|
|`PFLT_PYXIS_HOST`|env|The Pyxis host to connect to. Must contain any additional path information leading up to the API version|optional|catalog.redhat.com/api/containers|
|`PFLT_PYXIS_API_TOKEN`|env|The API Token to be used when connecting to Pyxis. Used for authenticated calls only.|optional?|-|
|`PFLT_DOCKERCONFIG`|env|The full path to a dockerconfigjson file, that has access to the container under test.|required|-|
