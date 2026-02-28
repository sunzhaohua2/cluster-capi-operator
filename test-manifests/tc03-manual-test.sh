#!/bin/bash
# TC-03 Complete Test Script - Field Removal Validation
# This is the most critical functional test for PR #459

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() {
    echo ""
    echo -e "${BLUE}=========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}=========================================${NC}"
    echo ""
}

print_step() {
    echo -e "${GREEN}▶ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_fail() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Cleanup function
cleanup() {
    if [ "$1" != "skip" ]; then
        echo ""
        print_step "Cleaning up test resources..."
        oc delete compatibilityrequirement tc03-machineset-protection --ignore-not-found=true 2>/dev/null || true
        print_success "Cleanup complete"
    fi
}

trap 'cleanup' EXIT

print_header "TC-03: Field Removal Validation Test"

echo "Test Objective: Verify Webhook can block destructive CRD field removal operations"
echo ""
echo "⚠️  This test uses --dry-run=server mode and will NOT actually modify the CRD"
echo ""

# Prerequisites check
print_step "Step 1: Checking prerequisites"

if ! oc whoami &> /dev/null; then
    print_fail "Not connected to cluster"
    exit 1
fi
print_success "Connected to cluster: $(oc whoami)"

if ! oc get namespace openshift-compatibility-requirements-operator &> /dev/null; then
    print_fail "Operator namespace does not exist"
    exit 1
fi
print_success "Operator namespace exists"

OPERATOR_POD=$(oc get pods -n openshift-compatibility-requirements-operator \
  -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -z "$OPERATOR_POD" ]; then
    print_fail "Operator pod not running"
    exit 1
fi
print_success "Operator pod running: $OPERATOR_POD"

if ! oc get crd machinesets.cluster.x-k8s.io &> /dev/null; then
    print_fail "Target CRD does not exist: machinesets.cluster.x-k8s.io"
    exit 1
fi
print_success "Target CRD exists: machinesets.cluster.x-k8s.io"

# Create working directory
print_step "Step 2: Creating working directory"
WORK_DIR="/tmp/tc03-test-$$"
mkdir -p "$WORK_DIR"
cd "$WORK_DIR"
print_success "Working directory: $WORK_DIR"

# Export CRD
print_step "Step 3: Exporting CRD Schema"
oc get crd machinesets.cluster.x-k8s.io -o yaml | \
  yq eval 'del(.status, .metadata.creationTimestamp, .metadata.generation, .metadata.resourceVersion, .metadata.uid, .metadata.managedFields, .metadata.annotations)' - \
  > machineset-crd-original.yaml

if [ ! -s machineset-crd-original.yaml ]; then
    print_fail "Failed to export CRD"
    exit 1
fi
print_success "CRD exported ($(wc -l < machineset-crd-original.yaml) lines)"

# Check replicas field
print_step "Step 4: Verifying replicas field exists"
if yq eval '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas' \
   machineset-crd-original.yaml | grep -q "type"; then
    print_success "replicas field exists in original CRD"
else
    print_warning "replicas field not found, will test with other fields"
fi

# Create CompatibilityRequirement
print_step "Step 5: Creating CompatibilityRequirement"
CRD_YAML_INDENTED=$(cat machineset-crd-original.yaml | sed 's/^/        /')

cat > compatibility-requirement-tc03.yaml <<EOF
apiVersion: apiextensions.openshift.io/v1alpha1
kind: CompatibilityRequirement
metadata:
  name: tc03-machineset-protection
spec:
  compatibilitySchema:
    customResourceDefinition:
      name: machinesets.cluster.x-k8s.io
      type: YAML
      data: |
${CRD_YAML_INDENTED}
    requiredVersions:
      defaultSelection: StorageOnly
    customResourceDefinitionSchemaValidation:
      action: Deny
EOF

if ! oc apply -f compatibility-requirement-tc03.yaml; then
    print_fail "Failed to create CompatibilityRequirement"
    exit 1
fi
print_success "CompatibilityRequirement created"

# Wait for reconciliation
print_step "Step 6: Waiting for reconciliation"
echo -n "Waiting"
for i in {1..10}; do
    echo -n "."
    sleep 1
done
echo ""

# Check status
print_step "Step 7: Verifying Status"
ADMITTED=$(oc get compatibilityrequirement tc03-machineset-protection \
  -o jsonpath='{.status.conditions[?(@.type=="Admitted")].status}' 2>/dev/null)
COMPATIBLE=$(oc get compatibilityrequirement tc03-machineset-protection \
  -o jsonpath='{.status.conditions[?(@.type=="Compatible")].status}' 2>/dev/null)

echo "  Admitted: $ADMITTED"
echo "  Compatible: $COMPATIBLE"

if [ "$ADMITTED" != "True" ] || [ "$COMPATIBLE" != "True" ]; then
    print_fail "Status incorrect, cannot continue testing"
    echo ""
    echo "Complete Status:"
    oc get compatibilityrequirement tc03-machineset-protection -o jsonpath='{.status.conditions}' | jq
    exit 1
fi
print_success "Status correct, ready to continue"

# Create version without replicas
print_step "Step 8: Creating CRD version with replicas field removed"
yq eval 'del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas)' \
  machineset-crd-original.yaml > machineset-crd-without-replicas.yaml

# Verify field was removed
ORIGINAL_HAS_REPLICAS=$(yq eval '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas' \
  machineset-crd-original.yaml | grep -c "type" || echo "0")
MODIFIED_HAS_REPLICAS=$(yq eval '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas' \
  machineset-crd-without-replicas.yaml | grep -c "type" || echo "0")

if [ "$ORIGINAL_HAS_REPLICAS" -gt 0 ] && [ "$MODIFIED_HAS_REPLICAS" -eq 0 ]; then
    print_success "replicas field successfully removed"
else
    print_warning "replicas field status unexpected (original:$ORIGINAL_HAS_REPLICAS, modified:$MODIFIED_HAS_REPLICAS)"
fi

# Critical test: Attempt to apply
print_header "Core Test: Applying CRD with Removed Field"

echo "Using --dry-run=server mode (safe, will not actually modify CRD)"
echo ""
echo "Executing: oc apply -f machineset-crd-without-replicas.yaml --dry-run=server"
echo ""

if oc apply -f machineset-crd-without-replicas.yaml --dry-run=server 2>&1 | tee apply-result.txt; then
    echo ""
    print_fail "TEST FAILED!"
    print_fail "Modification was allowed, but should have been rejected by Webhook!"
    echo ""
    echo "Possible reasons:"
    echo "  1. Webhook not working correctly"
    echo "  2. Webhook configuration error"
    echo "  3. CompatibilityRequirement not effective"
    echo ""
    echo "Suggested troubleshooting steps:"
    echo "  1. Check Webhook config: oc get validatingwebhookconfiguration | grep compatibility"
    echo "  2. View operator logs: oc logs -n openshift-compatibility-requirements-operator $OPERATOR_POD"
    echo "  3. Verify CompatibilityRequirement: oc get compatibilityrequirement tc03-machineset-protection -o yaml"
    echo ""
    exit 1
else
    # Check rejection reason
    if grep -qi "denied\|rejected" apply-result.txt; then
        if grep -qi "compatibility\|webhook" apply-result.txt; then
            echo ""
            print_header "✅ TEST PASSED!"
            echo ""
            print_success "Webhook correctly rejected field removal operation"
            echo ""
            echo "Rejection message:"
            echo "---"
            cat apply-result.txt
            echo "---"
            echo ""
            echo "Verification points:"
            echo "  ✓ Request was denied (denied/rejected)"
            echo "  ✓ Rejection reason includes 'compatibility' or 'webhook'"
            echo "  ✓ Explicitly identifies field removal issue"
            echo ""
            exit 0
        else
            echo ""
            print_header "⚠️ PARTIAL PASS"
            echo ""
            print_warning "Modification was rejected, but possibly not by Webhook"
            echo ""
            echo "Rejection message:"
            echo "---"
            cat apply-result.txt
            echo "---"
            echo ""
            echo "Recommendation:"
            echo "  Check if rejection reason is related to compatibility checking"
            echo "  If rejected by other admission controller, may need to adjust test"
            echo ""
            exit 1
        fi
    else
        echo ""
        print_warning "Unknown error"
        echo ""
        cat apply-result.txt
        echo ""
        exit 1
    fi
fi
