---
name: preflight-container-check
description: Guide users running preflight check container for Red Hat Software Certification. Use when users want to validate containers, debug check failures, understand certification requirements, interpret results/artifacts/logs, run specific checks, or submit to Red Hat. Trigger on mentions of preflight, container validation, certification, check failures, or Red Hat Partner Connect.
---

# Preflight Container Check Skill

Help end users run `preflight check container` to validate their containers for Red Hat Software Certification. This skill assists with running checks, debugging failures, interpreting results, and submitting to Red Hat.

## When to Use This Skill

Use this skill when users:
- Want to validate a container image for Red Hat certification
- Need help understanding check failures or errors
- Ask about specific checks (HasLicense, BasedOnUbi, RunsAsNonroot, etc.)
- Need to interpret results, artifacts, or log files
- Want to submit results to Red Hat Partner Connect
- Have authentication or registry issues
- Are working in disconnected/offline environments
- Need multi-architecture validation

## Core Command Structure

The basic command structure is:

```bash
preflight check container <image-reference> [flags]
```

### Common Usage Patterns

**Basic validation (iterative testing):**
```bash
preflight check container quay.io/repo-name/container-name:version
```

**With authentication:**
```bash
preflight check container quay.io/repo-name/container-name:version \
  --docker-config=/path/to/config.json
```

**Submitting results to Red Hat:**
```bash
preflight check container quay.io/repo-name/container-name:version \
  --submit \
  --pyxis-api-token=$PFLT_PYXIS_API_TOKEN \
  --certification-component-id=$PFLT_CERTIFICATION_COMPONENT_ID \
  --docker-config=/path/to/config.json
```

**Testing local images (before pushing):**
```bash
# Start a local registry
podman run -p 5000:5000 docker.io/library/registry

# Push to local registry
podman push --tls-verify=false localhost/myrepo/mycontainer:v1.0 localhost:5000/myrepo/mycontainer:v1.0

# Run preflight
preflight check container --insecure localhost:5000/myrepo/mycontainer:v1.0
```
Note: `--submit` and `--insecure` are mutually exclusive - cannot submit from insecure registry.

**Offline mode (disconnected environments):**
```bash
preflight check container quay.io/repo-name/container-name:version \
  --offline \
  --docker-config=/path/to/config.json
```
This creates a tarball of artifacts for submission by other Red Hat tools.

## Important Flags and Environment Variables

### Authentication
- `--docker-config` / `PFLT_DOCKERCONFIG`: Path to docker config.json (strongly recommended even for public Docker Hub images due to rate limits)

### Submission
- `--submit` / `-s`: Submit results to Red Hat (requires API token and component ID)
- `--pyxis-api-token` / `PFLT_PYXIS_API_TOKEN`: API token from Red Hat Partner Connect
- `--certification-component-id` / `PFLT_CERTIFICATION_COMPONENT_ID`: Component ID from connect.redhat.com
- `--pyxis-env`: Environment for submissions (default: "prod")

### Registry Options
- `--insecure`: Use insecure protocol (cannot be used with `--submit`)
- `--offline`: Generate artifacts tarball for later submission (cannot be used with `--submit`)
- `--platform`: Architecture to pull (default: runtime platform - amd64, arm64, ppc64le, s390x)

### Output and Logging
- `--artifacts` / `PFLT_ARTIFACTS`: Where artifacts are written (default: `artifacts/`)
- `--logfile` / `PFLT_LOGFILE`: Execution log location (default: `preflight.log`)
- `--loglevel` / `PFLT_LOGLEVEL`: Verbosity (warn, info, debug, trace, error)

## Understanding Check Results

### Results Location
After running preflight, check these locations:
- **Results JSON**: `artifacts/results.json` - detailed pass/fail for each check
- **Log file**: `preflight.log` (or custom location) - execution details
- **Artifacts**: `artifacts/` directory - check-specific evidence and data

### Common Container Checks

When debugging failures, understand what each check validates:

| Check Name | Purpose | Common Failures |
|------------|---------|-----------------|
| **HasLicense** | Verifies license files exist in standard locations | Missing LICENSE, COPYING, or copyright files |
| **BasedOnUbi** | Ensures container uses Red Hat Universal Base Image | Not built FROM a UBI base image |
| **HasRequiredLabels** | Checks for required container labels | Missing name, vendor, version, release, summary, or description labels |
| **HasUniqueTag** | Validates tag is not 'latest' | Using :latest tag |
| **RunsAsNonroot** | Ensures container doesn't run as root | USER directive missing or set to root/UID 0 |
| **HasModifiedFiles** | Checks if RPM files were modified | Files from RPM packages have been altered |
| **MaxLayers** | Validates layer count is reasonable | Too many layers (increases attack surface) |
| **HasProhibitedPackages** | Checks for prohibited software | Contains packages not allowed in certified containers |

### Interpreting Failures

When a check fails:

1. **Read the result message**: The `results.json` contains a `message` field explaining why
2. **Check artifacts**: Look in `artifacts/<platform>/` for check-specific evidence
3. **Review logs**: Use `--loglevel=debug` or `--loglevel=trace` for detailed execution info
4. **Understand the requirement**: Each check enforces Red Hat certification policy

**Example: Debugging a HasLicense failure**

```bash
# Run with debug logging
preflight check container quay.io/myrepo/myimage:v1.0 --loglevel=debug

# Check the results
cat artifacts/results.json | jq '.results[] | select(.check == "HasLicense")'

# Common fixes:
# - Add LICENSE or COPYING file to image
# - Ensure license file is in /licenses/, /LICENSE, or root directory
```

## Multi-Architecture Validation

Preflight automatically detects manifest lists and processes all supported architectures (amd64, arm64, ppc64le, s390x):

```bash
# Processes all architectures in manifest list
preflight check container quay.io/repo/multi-arch:v1.0

# Process only specific architecture
preflight check container quay.io/repo/multi-arch:v1.0 --platform=arm64
```

Results are organized by platform: `artifacts/amd64/`, `artifacts/arm64/`, etc.

## Configuration File

Avoid exposing tokens in console by using a config file:

```yaml
# config.yaml
dockerConfig: path/to/config.json
loglevel: trace
logfile: artifacts/preflight.log
artifacts: artifacts
junit: true
certification_component_id: your_component_id
pyxis_api_token: your_token
```

Then run:
```bash
preflight check container your-image:tag --submit
```

## Getting Certification Credentials

**Certification Component ID:**
1. Go to connect.redhat.com and navigate to your component
2. Look at the Overview page URL: `https://connect.redhat.com/component/{certification-component-id}/images`
3. The ID is the value between `/component/` and `/images`
4. May differ from the component PID shown on the overview page
5. Use without the `ospid-` prefix if present

**API Token:**
1. Visit https://connect.redhat.com/account/api-keys
2. Create a new API key
3. Copy the token value

## Common Workflows

### Iterative Development
```bash
# 1. Build your container
podman build -t myimage:v1.0 .

# 2. Push to registry
podman push myimage:v1.0 quay.io/myrepo/myimage:v1.0

# 3. Run preflight (no submission)
preflight check container quay.io/myrepo/myimage:v1.0

# 4. Fix failures, rebuild, repeat until all checks pass

# 5. Once passing, submit
preflight check container quay.io/myrepo/myimage:v1.0 \
  --submit \
  --pyxis-api-token=$PFLT_PYXIS_API_TOKEN \
  --certification-component-id=$PFLT_CERTIFICATION_COMPONENT_ID
```

### CI/CD Integration
```bash
# Test before pushing to public registry
podman run -d -p 5000:5000 --name registry docker.io/library/registry
podman push --tls-verify=false localhost/myimage:v1.0 localhost:5000/myimage:v1.0
preflight check container --insecure localhost:5000/myimage:v1.0

# If checks pass, push to public registry and submit
if [ $? -eq 0 ]; then
  podman push myimage:v1.0 quay.io/myrepo/myimage:v1.0
  preflight check container quay.io/myrepo/myimage:v1.0 --submit ...
fi
```

## Troubleshooting Common Issues

### Authentication Errors
```text
Error: failed to pull image: unauthorized
```
**Fix:** Provide docker config with credentials
```bash
podman login --username [USERNAME] --password [PASSWORD] --authfile ./auth.json [REGISTRY]
preflight check container <image> --docker-config=./auth.json
```

### Rate Limiting (Docker Hub)
```text
Error: rate limit exceeded
```
**Fix:** Even for public images, provide authenticated docker config to avoid Docker Hub rate limits

### Missing Certification Component ID
```text
Error: certification component ID must be specified when --submit is present
```
**Fix:** Provide component ID from Partner Connect:
```bash
--certification-component-id=1234567890abc
```

### Platform Mismatch
```text
Error: cannot process image manifest of different arch without platform override
```
**Fix:** Specify the platform explicitly:
```bash
--platform=amd64
```

### Offline Submission
If running in a disconnected environment, use `--offline` to create an artifacts tarball:
```bash
preflight check container <image> --offline
```
This creates `artifacts.tar.gz` containing all artifacts for submission via other Red Hat tools.

## Tips for Success

1. **Start without --submit**: Test iteratively until all checks pass, then submit
2. **Use debug logging**: When debugging, use `--loglevel=debug` or `--loglevel=trace`
3. **Understand the policies**: Review Red Hat certification requirements at https://access.redhat.com/documentation/en-us/red_hat_software_certification
4. **Check artifacts**: The `artifacts/` directory contains valuable debugging information
5. **Multi-arch testing**: If shipping multi-arch images, test all platforms before submitting
6. **Authentication**: Always provide docker config, even for public images (rate limits)
7. **Review results.json**: This file contains detailed pass/fail information for each check

## When to Escalate

If users encounter issues beyond this skill's scope:
- **Build failures**: This skill covers using preflight, not building the preflight tool itself
- **Check implementation bugs**: Report issues at https://github.com/redhat-openshift-ecosystem/openshift-preflight/issues
- **Certification policy questions**: Direct to Red Hat certification documentation or Partner Connect support at https://connect.redhat.com/support/partner-acceleration-desk/#/case/new
- **API/Pyxis issues**: Contact Red Hat Partner Connect support at https://connect.redhat.com/support/partner-acceleration-desk/#/case/new

## Example: Complete Submission Workflow

```bash
# Set up credentials
export PFLT_PYXIS_API_TOKEN="your-api-token"
export PFLT_CERTIFICATION_COMPONENT_ID="your-component-id"

# Create docker config for authentication
podman login --username myuser --password mypass --authfile ./config.json quay.io

# Run validation (iterative)
preflight check container quay.io/myrepo/myimage:v1.0 \
  --docker-config=./config.json \
  --loglevel=debug

# Review results
cat artifacts/results.json | jq '.passed'

# If all checks pass, submit
preflight check container quay.io/myrepo/myimage:v1.0 \
  --submit \
  --docker-config=./config.json

# Verify submission succeeded
echo "Check Partner Connect for submission status"
```
