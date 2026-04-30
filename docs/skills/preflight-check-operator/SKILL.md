---
name: preflight-check-operator
description: Guide users running preflight check operator for Red Hat OpenShift Operator Certification. Use when users want to validate operator bundles, debug check failures, understand OLM deployment issues, interpret scorecard results, build index images, or work with operator certification requirements. Trigger on mentions of preflight operator, operator bundle, OLM, scorecard, DeployableByOLM, operator certification, or index images.
---

# Preflight Operator Check Skill

Help end users run `preflight check operator` to validate their operator bundles for Red Hat OpenShift Operator Certification. This skill assists with running checks, building index images, debugging failures, interpreting scorecard results, and understanding operator certification requirements.

## When to Use This Skill

Use this skill when users:
- Want to validate an operator bundle for Red Hat OpenShift certification
- Need help building or configuring index images with `opm`
- Need to understand check failures or errors (especially DeployableByOLM, Scorecard)
- Ask about specific operator checks (DeployableByOLM, ValidateOperatorBundle, CertifiedImages, etc.)
- Need to interpret results, artifacts, or log files from operator validation
- Have issues with KUBECONFIG, cluster access, or OLM deployment
- Are working in disconnected/air-gapped environments with operator bundles
- Need to configure operator channels, namespaces, or service accounts

## Core Command Structure

The basic command structure is:

```bash
preflight check operator <bundle-image-reference> [flags]
```

**Critical Prerequisites:**
- A running OpenShift 4.5+ cluster with Operator Lifecycle Manager (OLM)
- KUBECONFIG environment variable pointing to cluster with cluster-admin privileges
- An index image containing your operator bundle
- Your operator bundle published to a container registry

## Common Usage Patterns

**Basic operator validation:**
```bash
export KUBECONFIG=/path/to/your/kubeconfig
export PFLT_INDEXIMAGE=registry.example.org/your-namespace/your-index:v1.0
preflight check operator registry.example.org/your-namespace/your-bundle:v1.0
```

**With authentication for private registries:**
```bash
export KUBECONFIG=/path/to/your/kubeconfig
export PFLT_INDEXIMAGE=registry.example.org/your-namespace/your-index:v1.0
preflight check operator registry.example.org/your-namespace/your-bundle:v1.0 \
  --docker-config=/path/to/config.json
```

**Specifying operator channel:**
```bash
export KUBECONFIG=/path/to/your/kubeconfig
export PFLT_INDEXIMAGE=registry.example.org/your-namespace/your-index:v1.0
preflight check operator registry.example.org/your-namespace/your-bundle:v1.0 \
  --channel=beta
```

**Custom namespace and service account:**
```bash
export KUBECONFIG=/path/to/your/kubeconfig
export PFLT_INDEXIMAGE=registry.example.org/your-namespace/your-index:v1.0
preflight check operator registry.example.org/your-namespace/your-bundle:v1.0 \
  --namespace=preflight-testing \
  --serviceaccount=preflight-sa
```

**Disconnected/air-gapped environment:**
```bash
# First, get the scorecard image digest on a connected machine
preflight runtime-assets

# Mirror the scorecard image to your disconnected registry
# Then use it in disconnected environment
export KUBECONFIG=/path/to/your/kubeconfig
export PFLT_INDEXIMAGE=registry.internal/operators/your-index:v1.0
preflight check operator registry.internal/operators/your-bundle:v1.0 \
  --scorecard-image=registry.internal/scorecard@sha256:abc123... \
  --docker-config=/path/to/config.json
```

## Important Flags and Environment Variables

### Required
- `KUBECONFIG` (env): Path to kubeconfig with cluster-admin access to OpenShift 4.5+ cluster
- `PFLT_INDEXIMAGE` (env): Index image containing your operator bundle

### Authentication
- `--docker-config` / `PFLT_DOCKERCONFIG`: Path to docker config.json (for private registries or to avoid rate limits)

### Operator Configuration
- `--channel` / `PFLT_CHANNEL`: Operator channel name for DeployableByOLM check (uses default channel from bundle annotations if empty)
- `--namespace` / `PFLT_NAMESPACE`: Namespace for OperatorSDK Scorecard execution (default: "default")
- `--serviceaccount` / `PFLT_SERVICEACCOUNT`: Service account for OperatorSDK Scorecard (default: "default")
- `--scorecard-wait-time` / `PFLT_SCORECARD_WAIT_TIME`: Time value passed to scorecard's --wait-time

### Disconnected Environments
- `--scorecard-image` / `PFLT_SCORECARD_IMAGE`: URI pointing to scorecard image digest (for disconnected environments only)

### Output and Logging
- `--artifacts` / `PFLT_ARTIFACTS`: Where artifacts are written (default: `artifacts/`)
- `--logfile` / `PFLT_LOGFILE`: Execution log location (default: `preflight.log`)
- `--loglevel` / `PFLT_LOGLEVEL`: Verbosity (warn, info, debug, trace, error)

## Building an Index Image

Before running operator checks, you **must** build an index image containing your bundle. This is required for the DeployableByOLM check.

### Prerequisites
1. Install the `opm` CLI tool from [operator-framework/operator-registry releases](https://github.com/operator-framework/operator-registry/releases)
2. Your operator bundle must already be published to a registry

### Building the Index

```bash
# Build index image with your bundle
opm index add \
  --bundles registry.example.org/your-namespace/your-bundle:v1.0 \
  --tag registry.example.org/your-namespace/your-index:v1.0

# Push the index to your registry
podman push registry.example.org/your-namespace/your-index:v1.0

# Set the index image for preflight
export PFLT_INDEXIMAGE=registry.example.org/your-namespace/your-index:v1.0
```

**Docker users:** Add `--container-tool=docker` to the `opm index add` command.

**Private registries:** If your index image is in a private repository, provide docker config:
```bash
export PFLT_DOCKERCONFIG=/path/to/docker/config.json
```

### Index Image Requirements
- Must be accessible from both preflight and your target OpenShift cluster
- Must contain the operator bundle you're testing
- Registry must be reachable from the cluster (important for DeployableByOLM)

## Understanding Check Results

### Results Location
After running preflight, check these locations:
- **Results JSON**: `artifacts/results.json` - detailed pass/fail for each check
- **Log file**: `preflight.log` (or custom location) - execution details
- **Artifacts**: `artifacts/` directory - check-specific evidence and scorecard output

### Common Operator Checks

When debugging failures, understand what each check validates:

| Check Name | Purpose | Common Failures |
|------------|---------|-----------------|
| **DeployableByOLM** | Validates operator can be deployed via OLM | Index image not accessible, CSV issues, missing dependencies, cluster connectivity |
| **ValidateOperatorBundle** | Runs `operator-sdk bundle validate` | Invalid bundle structure, missing required files, annotation errors |
| **ScorecardBasicSpecCheck** | Runs OperatorSDK Scorecard basic tests | Operator doesn't install properly, status not updated, CRD validation issues |
| **ScorecardOlmSuiteCheck** | Runs OperatorSDK Scorecard OLM suite | OLM-specific issues, bundle metadata problems, CSV validation failures |
| **CertifiedImages** | Verifies all container images are Red Hat certified | Using non-certified base images or dependencies |
| **RequiredAnnotations** | Checks for required bundle annotations | Missing annotations in metadata/annotations.yaml |
| **RelatedImages** | Validates relatedImages in CSV | Missing or incorrect relatedImages section in ClusterServiceVersion |
| **RestrictedNetworkAware** | Checks for disconnected environment support | RelatedImages not properly declared, external image references |

### Interpreting Failures

When a check fails:

1. **Read the result message**: The `results.json` contains a `message` field with details
2. **Check artifacts**: Look in `artifacts/` for check-specific evidence (scorecard output, logs)
3. **Review cluster logs**: For DeployableByOLM failures, check cluster events and pod logs
4. **Use debug logging**: Run with `--loglevel=debug` or `--loglevel=trace`
5. **Verify prerequisites**: Confirm KUBECONFIG, PFLT_INDEXIMAGE, and cluster access

## Debugging Common Failures

### DeployableByOLM Failures

This is the most common failure point. The check deploys your operator on the cluster using OLM.

**Common issues:**

1. **Index image not accessible from cluster**
   ```
   Error: Failed to create CatalogSource
   ```
   **Fix:** Ensure the index image is in a registry accessible from the cluster. For private registries, create an image pull secret:
   ```bash
   # Provide docker config to preflight
   preflight check operator <bundle> --docker-config=/path/to/config.json
   ```

2. **ClusterServiceVersion (CSV) issues**
   ```
   Error: CSV did not reach succeeded phase
   ```
   **Fix:** Check the CSV definition in your bundle. Common problems:
   - Missing or incorrect install modes
   - Invalid deployment specifications
   - Resource requirements too high for test cluster
   - Missing RBAC permissions

3. **Missing dependencies**
   ```
   Error: Required CRDs not found
   ```
   **Fix:** Ensure all CRD dependencies are declared in the CSV and available in the cluster.

4. **Channel configuration**
   ```text
   Error: Channel not found in bundle
   ```
   **Fix:** Either specify `--channel=<channel-name>` or ensure your bundle has a default channel in `metadata/annotations.yaml`.

**Debugging steps:**
```bash
# Run with debug logging
preflight check operator <bundle> --loglevel=debug

# Check the operator pod logs in the cluster
oc get pods -n <namespace>
oc logs <operator-pod> -n <namespace>

# Check the subscription and CSV
oc get subscription -n <namespace>
oc get csv -n <namespace>
oc describe csv <csv-name> -n <namespace>
```

### Scorecard Failures

Scorecard checks run OperatorSDK tests against your operator.

**Common issues:**

1. **Timeout waiting for scorecard**
   ```
   Error: Scorecard timed out
   ```
   **Fix:** Increase wait time:
   ```bash
   preflight check operator <bundle> --scorecard-wait-time=300s
   ```

2. **Service account permissions**
   ```
   Error: ServiceAccount doesn't have required permissions
   ```
   **Fix:** Use a service account with appropriate RBAC:
   ```bash
   # Create service account with proper permissions
   oc create serviceaccount preflight-sa -n preflight-testing
   # Add necessary role bindings
   oc create rolebinding preflight-admin --clusterrole=admin --serviceaccount=preflight-testing:preflight-sa -n preflight-testing
   
   preflight check operator <bundle> \
     --namespace=preflight-testing \
     --serviceaccount=preflight-sa
   ```

3. **Scorecard image not available (disconnected)**
   ```
   Error: Failed to pull scorecard image
   ```
   **Fix:** Use `preflight runtime-assets` on connected machine, mirror the image, then:
   ```bash
   preflight check operator <bundle> --scorecard-image=<internal-registry>/scorecard@sha256:...
   ```

### ValidateOperatorBundle Failures

This check validates bundle structure using `operator-sdk bundle validate`.

**Common issues:**

1. **Missing required files**
   ```
   Error: Missing manifests directory
   ```
   **Fix:** Ensure bundle has proper structure:
   ```
   bundle/
   ├── manifests/
   │   ├── <operator>.clusterserviceversion.yaml
   │   └── <crd-files>.yaml
   └── metadata/
       └── annotations.yaml
   ```

2. **Invalid annotations**
   ```
   Error: Required annotation missing
   ```
   **Fix:** Verify `metadata/annotations.yaml` contains required fields:
   ```yaml
   annotations:
     operators.operatorframework.io.bundle.manifests.v1: manifests/
     operators.operatorframework.io.bundle.metadata.v1: metadata/
     operators.operatorframework.io.bundle.package.v1: <package-name>
     operators.operatorframework.io.bundle.channels.v1: <channel-list>
     operators.operatorframework.io.bundle.channel.default.v1: <default-channel>
   ```

### CertifiedImages Failures

Validates all images used by the operator are Red Hat certified.

```text
Error: Image <image-name> is not certified
```

**Fix:** All images referenced in your CSV must be from the Red Hat certified catalog. Check:
- Base images in Dockerfiles
- Images in CSV's relatedImages section
- Container images in deployment specs

**Where to find certified images:**
- Red Hat Container Catalog: https://catalog.redhat.com/
- Use Red Hat Universal Base Images (UBI) as base images
- Ensure all sidecar/init containers use certified images

### RelatedImages / RestrictedNetworkAware Failures

These checks ensure operator works in disconnected environments.

```text
Error: RelatedImages section is incomplete
```

**Fix:** The CSV must have a complete `relatedImages` section listing all images:
```yaml
spec:
  relatedImages:
    - name: operator
      image: registry.example.org/operator:v1.0.0
    - name: operand
      image: registry.example.org/operand:v1.0.0
    # Include ALL images referenced by the operator
```

**Important:** Every image used must be in relatedImages, including:
- Operator image itself
- All operand images
- Init containers
- Sidecar containers
- Any images referenced in the operator's code

## Configuration File

Use a config file to avoid exposing values in the console:

```yaml
# config.yaml
dockerConfig: /path/to/config.json
loglevel: debug
logfile: artifacts/preflight.log
artifacts: artifacts
namespace: preflight-testing
serviceaccount: preflight-sa
channel: stable
scorecard_wait_time: 300s
```

Then set environment variables and run:
```bash
export KUBECONFIG=/path/to/kubeconfig
export PFLT_INDEXIMAGE=registry.example.org/your-namespace/your-index:v1.0
preflight check operator registry.example.org/your-namespace/your-bundle:v1.0
```

## Common Workflows

### Iterative Development Workflow

```bash
# 1. Build and push your operator bundle
operator-sdk bundle build quay.io/myrepo/my-operator-bundle:v1.0
podman push quay.io/myrepo/my-operator-bundle:v1.0

# 2. Build and push index image
opm index add --bundles quay.io/myrepo/my-operator-bundle:v1.0 \
  --tag quay.io/myrepo/my-operator-index:v1.0
podman push quay.io/myrepo/my-operator-index:v1.0

# 3. Set up environment
export KUBECONFIG=/path/to/kubeconfig
export PFLT_INDEXIMAGE=quay.io/myrepo/my-operator-index:v1.0

# 4. Run preflight (iterative - no submission for operators)
preflight check operator quay.io/myrepo/my-operator-bundle:v1.0 --loglevel=debug

# 5. Fix failures, rebuild bundle and index, repeat until all checks pass

# 6. Review results
cat artifacts/results.json | jq '.passed'
```

### Testing with Different Channels

```bash
# Test the stable channel
export PFLT_INDEXIMAGE=quay.io/myrepo/my-operator-index:v1.0
preflight check operator quay.io/myrepo/my-operator-bundle:v1.0 --channel=stable

# Test the beta channel
preflight check operator quay.io/myrepo/my-operator-bundle:v1.0 --channel=beta
```

### Podman Container Workflow

Running preflight in a container for operator checks:

```bash
CONTAINER_TOOL=podman
$CONTAINER_TOOL run \
  -it \
  --rm \
  --security-opt=label=disable \
  --env KUBECONFIG=/kubeconfig \
  --env PFLT_LOGLEVEL=debug \
  --env PFLT_INDEXIMAGE=registry.example.org/your-namespace/your-index:v1.0 \
  --env PFLT_ARTIFACTS=/artifacts \
  --env PFLT_CHANNEL=stable \
  --env PFLT_LOGFILE=/artifacts/preflight.log \
  -v /path/on/host/artifacts:/artifacts \
  -v /path/on/host/kubeconfig:/kubeconfig:ro \
  quay.io/opdev/preflight:stable check operator registry.example.org/your-namespace/your-bundle:v1.0
```

## Disconnected/Air-Gapped Environments

### Step 1: Get Required Images (on connected machine)

```bash
# Get the scorecard image digest
preflight runtime-assets
```

This outputs the scorecard image and other runtime assets that need to be mirrored.

### Step 2: Mirror Images

Mirror these images to your internal registry:
- Scorecard image
- Your operator bundle
- Your index image
- All images referenced in your operator's CSV

### Step 3: Run Preflight (in disconnected environment)

```bash
export KUBECONFIG=/path/to/kubeconfig
export PFLT_INDEXIMAGE=registry.internal/operators/my-index:v1.0
preflight check operator registry.internal/operators/my-bundle:v1.0 \
  --scorecard-image=registry.internal/scorecard@sha256:abc123... \
  --docker-config=/path/to/config.json \
  --loglevel=debug
```

## Cluster Requirements

### Minimum Requirements
- OpenShift 4.5 or later
- Operator Lifecycle Manager (OLM) installed and running
- Cluster-admin privileges via KUBECONFIG
- Sufficient resources to deploy the operator

### Verifying Cluster Access

```bash
# Verify KUBECONFIG is set and valid
echo $KUBECONFIG
oc cluster-info

# Verify OLM is running
oc get csv -A
oc get catalogsources -n openshift-marketplace

# Verify you have cluster-admin privileges
oc auth can-i '*' '*' --all-namespaces
```

### Namespace and RBAC Considerations

By default, preflight uses the `default` namespace and `default` service account. For isolated testing:

```bash
# Create dedicated namespace
oc create namespace preflight-testing

# Create service account
oc create serviceaccount preflight-sa -n preflight-testing

# Grant necessary permissions
oc create rolebinding preflight-admin \
  --clusterrole=admin \
  --serviceaccount=preflight-testing:preflight-sa \
  -n preflight-testing

# Run preflight with custom namespace/SA
preflight check operator <bundle> \
  --namespace=preflight-testing \
  --serviceaccount=preflight-sa
```

## Tips for Success

1. **Build index image first**: Don't forget this critical step - PFLT_INDEXIMAGE is required
2. **Verify cluster access**: Test KUBECONFIG and cluster connectivity before running checks
3. **Use debug logging**: For DeployableByOLM failures, `--loglevel=debug` shows cluster interaction
4. **Check cluster resources**: Ensure cluster has enough resources for operator deployment
5. **Test channels**: If your operator supports multiple channels, test each one
6. **Monitor cluster**: Watch pods/events in the cluster while DeployableByOLM runs
7. **Disconnected prep**: For air-gapped environments, use `preflight runtime-assets` to identify all required images
8. **Clean between runs**: If re-running checks, clean up operator resources from previous runs
9. **Review CSV carefully**: Most failures trace back to CSV configuration issues
10. **Complete relatedImages**: Ensure ALL images are listed for disconnected environment support

## Differences from Container Checks

Operator checks differ from container checks:
- **No submission**: Operator results are not submitted to Red Hat (submits to Partner Connect portal separately)
- **Requires cluster**: Must have access to OpenShift cluster with OLM
- **Requires index image**: Must build index image containing bundle
- **Tests deployment**: DeployableByOLM actually deploys the operator on the cluster
- **Scorecard integration**: Runs OperatorSDK Scorecard tests
- **More complex**: Operator certification has more moving parts than container certification

## When to Escalate

If users encounter issues beyond this skill's scope:
- **OLM/Cluster issues**: Direct to OpenShift documentation or support
- **Operator SDK problems**: Reference operator-framework documentation
- **Preflight bugs**: Report at https://github.com/redhat-openshift-ecosystem/openshift-preflight/issues
- **Certification policy questions**: Direct to Red Hat certification documentation or Partner Connect support at https://connect.redhat.com/support/partner-acceleration-desk/#/case/new
- **API/Pyxis issues**: Contact Red Hat Partner Connect support at https://connect.redhat.com/support/partner-acceleration-desk/#/case/new

## Example: Complete Validation Workflow

```bash
# Prerequisites check
echo "Checking prerequisites..."
which opm || echo "ERROR: opm not found"
oc cluster-info || echo "ERROR: Cluster not accessible"
echo $PFLT_INDEXIMAGE || echo "ERROR: PFLT_INDEXIMAGE not set"

# Build index image
echo "Building index image..."
opm index add \
  --bundles quay.io/myrepo/my-operator-bundle:v1.0.0 \
  --tag quay.io/myrepo/my-operator-index:v1.0.0

podman push quay.io/myrepo/my-operator-index:v1.0.0

# Set environment
export KUBECONFIG=/path/to/kubeconfig
export PFLT_INDEXIMAGE=quay.io/myrepo/my-operator-index:v1.0.0

# Run preflight with debug logging
echo "Running preflight operator checks..."
preflight check operator quay.io/myrepo/my-operator-bundle:v1.0.0 \
  --loglevel=debug \
  --channel=stable \
  --namespace=preflight-testing

# Review results
echo "Checking results..."
cat artifacts/results.json | jq '.passed'

if [ $(cat artifacts/results.json | jq '.passed') = "true" ]; then
  echo "SUCCESS: All operator checks passed!"
  echo "Next: Submit to Partner Connect portal for certification"
else
  echo "FAILED: Review artifacts/results.json for failures"
  cat artifacts/results.json | jq '.results[] | select(.passed == false)'
fi
```
