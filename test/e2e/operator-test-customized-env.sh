#!/usr/bin/env bash

set -e

DEFAULT_PREFLIGHT_BIN="preflight"
DEFAULT_OPERATOR_BUNDLE="quay.io/opdev/simple-demo-operator-bundle:latest"
DEFAULT_OPERATOR_INDEXIMAGE="quay.io/opdev/simple-demo-operator-catalog:latest"

PREFLIGHT_BIN=${PREFLIGHT_BIN:-"$DEFAULT_PREFLIGHT_BIN"}
OPERATOR_BUNDLE=${OPERATOR_BUNDLE:-"$DEFAULT_OPERATOR_BUNDLE"}
OPERATOR_INDEXIMAGE=${OPERATOR_INDEXIMAGE:-"$DEFAULT_OPERATOR_INDEXIMAGE"}

# This test sets preflight's environment variables to ensure that
# expectations as to what they mean are consistent.
customized_sa="customsa" \
customized_ns="customns" \
customized_logfile="sgol.txt" \
customized_artifacts="e2e_test_artifacts" \
results_file="./${customized_artifacts}/results.json"


 USAGE=" 
 Usage: 
   $(basename "$0")

   This script will run the preflight binary against an operator test. To be used in e2e testing.
   
   Environment variables required by preflight that are not otherwise reflected here are still
   required (e.g. KUBECONFIG), and must be set in the environment prior to running this script.
  
 Environment variables: 
   PREFLIGHT_BIN:           Specify the path to the compiled preflight binary.
                            If this is not provided, this script will execute the
                            preflight binary relative to the current working directory.
                            Default: "./$DEFAULT_PREFLIGHT_BIN"
   OPERATOR_BUNDLE:         Specify the registry path of the operator bundle under test.
                            Default: $DEFAULT_OPERATOR_BUNDLE
   OPERATOR_INDEXIMAGE      Specify the index image containing the operator bundle under
                            test.
                            Default: $DEFAULT_OPERATOR_INDEXIMAGE
 "

# This script doesn't take positional arguments. If the user provided one,
# assume they are asking for help and print the usage statement.
 if [ $# -ne 0 ]; then 
     echo "$USAGE" 
     exit 1 
 fi 

# Emit the runtime values for this script stdout.
echo "Preflight binary value: $PREFLIGHT_BIN"
echo "Operator bundle being tested: $OPERATOR_BUNDLE"
echo "Operator index for test: $OPERATOR_INDEXIMAGE"

echo "Creating custom namespace and service account for tests"
kubebin=$(which oc 2>/dev/null || which kubectl 2>/dev/null)
if [ "$kubebin" == "" ]; then
    echo "ERR could not find kubectl or oc which is required for this test."
    exit 5
fi

cleanupCluster() {
    echo "running cleanup."
    "$kubebin" delete ns --force customns || true
}

"$kubebin" create ns customns \
    || { echo "ERR unable to create custom namespace for this test" &&  exit 6 ;}
"$kubebin" create serviceaccount --namespace customns customsa \
    || { echo "ERR unable to create custom namespace for this test" && cleanupCluster && exit 6 ;}

# Run preflight with customized env.
echo "Running preflight"
echo -e "========================"
PFLT_LOGLEVEL=trace \
PFLT_INDEXIMAGE="${OPERATOR_INDEXIMAGE}" \
PFLT_SERVICEACCOUNT="$customized_sa" \
PFLT_NAMESPACE="$customized_ns" \
PFLT_LOGFILE="$customized_logfile" \
PFLT_ARTIFACTS="$customized_artifacts" \
    "./${PREFLIGHT_BIN}" check operator "${OPERATOR_BUNDLE}"

echo -e "\n========================"

# Check to make sure that the customized artifacts directory exists
test -d "$customized_artifacts" \
    || { echo "ERR the customized artifacts directory wasn't created." \
        && cleanupCluster \
        && exit 7 ;}

# Check to make sure that the logfile we customized via environment exists
test -f "$customized_logfile" \
    || { echo "ERR the customized results file wasn't created in the customized artifacts directry." \
        && cleanupCluster \
        && exit 7 ;}

# Before we check the error count, make sure that it still
# exists at the expected path in the results.json
errors=$(jq -r .results.errors < $results_file)
if [ "$errors" == "null" ]; then
    echo "ERR results file did not contain errors at the expected location (.results.errors)."
    echo "It's impossible to determine if the tests all passed for this asset."
    
    exit 2
fi

# Check the error count to make sure it's zero.
errorcount=$(jq -r '.results.errors | length'  < $results_file)
if [ "$errorcount" -ne "0" ]; then
    echo "ERR preflight tests threw an error for this asset."
    echo "This asset should pass all checks."
    cleanupCluster
    exit 3
fi

# Before we check the failure count, make sure that it still
# exists at the expected path in the results.json
failures=$(jq -r .results.failed < $results_file)
if [ "$failures" == "null" ]; then
    echo "ERR results file did not contain failures at the expected location (.results.failed)."
    echo "It's impossible to determine if the tests all passed for this asset."
    cleanupCluster
    exit 4
fi

# Check the failure count to make sure it's zero.
failurecount=$(jq -r '.results.failed | length'  < $results_file)
if [ "$failurecount" -ne "0" ]; then
    echo "ERR preflight tests threw an error for this asset."
    echo "This asset should pass all checks."
    cleanupCluster
    exit 3
fi

echo "Everything appears to have passed!"
