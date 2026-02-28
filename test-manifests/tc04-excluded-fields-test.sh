#!/bin/bash

###############################################################################
# TC-04: Test ExcludedFields Functionality
#
# This script tests PR #467 - excludedFields feature
# Verifies that fields listed in excludedFields can be removed from CRD
###############################################################################

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
CRD_NAME="machinesets.cluster.x-k8s.io"
CR_NAME="tc04-machineset-excluded-fields"
NAMESPACE="openshift-cluster-api"
TMP_DIR="/tmp/tc04-test-$$"

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

###############################################################################
# Helper Functions
###############################################################################

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

section_header() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

cleanup() {
    log_info "Cleaning up test resources..."

    # Delete CompatibilityRequirement
    oc delete compatibilityrequirement "$CR_NAME" --ignore-not-found=true 2>/dev/null || true
    oc delete compatibilityrequirement "${CR_NAME}-multi" --ignore-not-found=true 2>/dev/null || true

    # Restore original CRD if backup exists
    if [[ -f "$TMP_DIR/machineset-crd-original.yaml" ]]; then
        log_info "Restoring original CRD..."
        oc apply -f "$TMP_DIR/machineset-crd-original.yaml" >/dev/null 2>&1 || true
    fi

    # Clean up temp directory
    rm -rf "$TMP_DIR"

    log_info "Cleanup complete"
}

# Register cleanup on exit
trap cleanup EXIT

###############################################################################
# Test Functions
###############################################################################

test_prerequisites() {
    section_header "Step 1: Checking Prerequisites"

    # Check oc command
    if ! command -v oc &> /dev/null; then
        log_error "oc command not found"
        return 1
    fi
    log_success "oc command available"

    # Check cluster connection
    if ! oc whoami &> /dev/null; then
        log_error "Not connected to OpenShift cluster"
        return 1
    fi
    log_success "Connected to cluster: $(oc whoami --show-server)"

    # Check if CRD exists
    if ! oc get crd "$CRD_NAME" &> /dev/null; then
        log_error "CRD $CRD_NAME not found"
        return 1
    fi
    log_success "CRD $CRD_NAME exists"

    # Check yq command
    if ! command -v yq &> /dev/null; then
        log_warning "yq not found - will use manual CRD editing"
    else
        log_success "yq command available"
    fi

    # Create temp directory
    mkdir -p "$TMP_DIR"
    log_success "Temp directory created: $TMP_DIR"
}

backup_original_crd() {
    section_header "Step 2: Backup Original CRD"

    log_info "Exporting current CRD..."
    oc get crd "$CRD_NAME" -o yaml > "$TMP_DIR/machineset-crd-original.yaml"

    # Verify replicas field exists
    if oc get crd "$CRD_NAME" -o jsonpath='{.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas}' | grep -q "type"; then
        log_success "Original CRD contains spec.replicas field"
    else
        log_warning "spec.replicas field not found in CRD - test may not be applicable"
    fi

    # Save original ResourceVersion
    ORIGINAL_RV=$(oc get crd "$CRD_NAME" -o jsonpath='{.metadata.resourceVersion}')
    log_info "Original ResourceVersion: $ORIGINAL_RV"
}

create_compatibility_requirement() {
    section_header "Step 3: Create CompatibilityRequirement with excludedFields"

    # Get storage version
    STORAGE_VERSION=$(oc get crd "$CRD_NAME" -o jsonpath='{.spec.versions[?(@.storage==true)].name}')
    log_info "Storage version: $STORAGE_VERSION"

    # Prepare indented CRD data
    oc get crd "$CRD_NAME" -o yaml | sed 's/^/        /' > "$TMP_DIR/crd-indented.yaml"

    # Create CompatibilityRequirement
    log_info "Creating CompatibilityRequirement with excludedFields..."

    cat <<EOF | oc apply -f -
apiVersion: apiextensions.openshift.io/v1alpha1
kind: CompatibilityRequirement
metadata:
  name: $CR_NAME
spec:
  compatibilitySchema:
    customResourceDefinition:
      name: $CRD_NAME
      type: YAML
      data: |
$(cat "$TMP_DIR/crd-indented.yaml")
    requiredVersions:
      defaultSelection: StorageOnly
    excludedFields:
      - path: "spec.replicas"
        versions:
          - $STORAGE_VERSION
  customResourceDefinitionSchemaValidation:
    action: Deny
EOF

    if [[ $? -eq 0 ]]; then
        log_success "CompatibilityRequirement created successfully"
    else
        log_error "Failed to create CompatibilityRequirement"
        return 1
    fi
}

verify_cr_status() {
    section_header "Step 4: Verify CompatibilityRequirement Status"

    log_info "Waiting for CompatibilityRequirement to be processed..."
    sleep 5

    # Check Admitted condition
    ADMITTED=$(oc get compatibilityrequirement "$CR_NAME" -o jsonpath='{.status.conditions[?(@.type=="Admitted")].status}' 2>/dev/null || echo "Unknown")
    if [[ "$ADMITTED" == "True" ]]; then
        log_success "CompatibilityRequirement Admitted=True"
    else
        log_error "CompatibilityRequirement Admitted=$ADMITTED"
        oc get compatibilityrequirement "$CR_NAME" -o jsonpath='{.status.conditions}' | jq .
        return 1
    fi

    # Check Compatible condition
    COMPATIBLE=$(oc get compatibilityrequirement "$CR_NAME" -o jsonpath='{.status.conditions[?(@.type=="Compatible")].status}' 2>/dev/null || echo "Unknown")
    if [[ "$COMPATIBLE" == "True" ]]; then
        log_success "CompatibilityRequirement Compatible=True"
    else
        log_warning "CompatibilityRequirement Compatible=$COMPATIBLE (may be expected)"
    fi
}

verify_excluded_fields() {
    section_header "Step 5: Verify excludedFields Configuration"

    log_info "Checking excludedFields configuration..."

    EXCLUDED_FIELDS=$(oc get compatibilityrequirement "$CR_NAME" \
        -o jsonpath='{.spec.compatibilitySchema.excludedFields}')

    if echo "$EXCLUDED_FIELDS" | grep -q "spec.replicas"; then
        log_success "excludedFields contains spec.replicas"
        echo "$EXCLUDED_FIELDS" | jq .
    else
        log_error "excludedFields does not contain spec.replicas"
        return 1
    fi
}

test_remove_excluded_field() {
    section_header "Step 6: Remove Excluded Field (spec.replicas)"

    log_info "This should SUCCEED because spec.replicas is in excludedFields"

    # Create modified CRD without spec.replicas
    if command -v yq &> /dev/null; then
        yq eval 'del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas)' \
            "$TMP_DIR/machineset-crd-original.yaml" > "$TMP_DIR/machineset-crd-no-replicas.yaml"
    else
        log_warning "yq not available - using sed (less reliable)"
        # This is a simplified approach - may need manual editing
        sed '/replicas:/,/type: integer/d' \
            "$TMP_DIR/machineset-crd-original.yaml" > "$TMP_DIR/machineset-crd-no-replicas.yaml"
    fi

    log_info "Applying CRD without spec.replicas field..."
    if oc apply -f "$TMP_DIR/machineset-crd-no-replicas.yaml" 2>&1 | tee "$TMP_DIR/apply-output.txt"; then
        log_success "✅ CRD update ALLOWED (spec.replicas removed)"

        # Verify ResourceVersion changed
        NEW_RV=$(oc get crd "$CRD_NAME" -o jsonpath='{.metadata.resourceVersion}')
        log_info "ResourceVersion changed: $ORIGINAL_RV → $NEW_RV"

        if [[ "$NEW_RV" != "$ORIGINAL_RV" ]]; then
            log_success "ResourceVersion updated (CRD was modified)"
        else
            log_warning "ResourceVersion unchanged (unexpected)"
        fi
    else
        log_error "❌ CRD update DENIED (should have been allowed)"
        cat "$TMP_DIR/apply-output.txt"
        return 1
    fi
}

verify_field_removed() {
    section_header "Step 7: Verify Field Was Actually Removed"

    REPLICAS_FIELD=$(oc get crd "$CRD_NAME" \
        -o jsonpath='{.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas}' 2>/dev/null || echo "")

    if [[ -z "$REPLICAS_FIELD" ]]; then
        log_success "spec.replicas field successfully removed from CRD"
    else
        log_error "spec.replicas field still exists in CRD"
        echo "Field content: $REPLICAS_FIELD"
        return 1
    fi
}

test_remove_non_excluded_field() {
    section_header "Step 8: Remove Non-Excluded Field (spec.selector)"

    log_info "This should FAIL because spec.selector is NOT in excludedFields"

    # First restore the CRD with replicas
    oc apply -f "$TMP_DIR/machineset-crd-original.yaml" >/dev/null 2>&1
    sleep 2

    # Create modified CRD without spec.selector
    if command -v yq &> /dev/null; then
        yq eval 'del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.selector)' \
            "$TMP_DIR/machineset-crd-original.yaml" > "$TMP_DIR/machineset-crd-no-selector.yaml"
    else
        log_warning "yq not available - skipping non-excluded field test"
        return 0
    fi

    log_info "Applying CRD without spec.selector field..."
    if oc apply -f "$TMP_DIR/machineset-crd-no-selector.yaml" 2>&1 | tee "$TMP_DIR/apply-selector-output.txt"; then
        log_error "❌ CRD update ALLOWED (should have been denied)"
        return 1
    else
        if grep -q "admission webhook.*denied" "$TMP_DIR/apply-selector-output.txt"; then
            log_success "✅ CRD update DENIED as expected (spec.selector not excluded)"
        else
            log_warning "CRD update failed but not due to webhook"
            cat "$TMP_DIR/apply-selector-output.txt"
        fi
    fi
}

test_multiple_excluded_fields() {
    section_header "Step 9: Test Multiple Excluded Fields"

    log_info "Creating CompatibilityRequirement with multiple excluded fields..."

    STORAGE_VERSION=$(oc get crd "$CRD_NAME" -o jsonpath='{.spec.versions[?(@.storage==true)].name}')

    cat <<EOF | oc apply -f -
apiVersion: apiextensions.openshift.io/v1alpha1
kind: CompatibilityRequirement
metadata:
  name: ${CR_NAME}-multi
spec:
  compatibilitySchema:
    customResourceDefinition:
      name: $CRD_NAME
      type: YAML
      data: |
$(cat "$TMP_DIR/crd-indented.yaml")
    requiredVersions:
      defaultSelection: StorageOnly
    excludedFields:
      - path: "spec.replicas"
      - path: "spec.minReadySeconds"
  customResourceDefinitionSchemaValidation:
    action: Deny
EOF

    if [[ $? -eq 0 ]]; then
        log_success "CompatibilityRequirement with multiple exclusions created"
    else
        log_error "Failed to create CompatibilityRequirement"
        return 1
    fi

    sleep 3

    # Try to remove both fields
    if command -v yq &> /dev/null; then
        log_info "Removing both spec.replicas and spec.minReadySeconds..."

        yq eval 'del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas) |
                 del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.minReadySeconds)' \
            "$TMP_DIR/machineset-crd-original.yaml" > "$TMP_DIR/machineset-crd-multi-excluded.yaml"

        if oc apply -f "$TMP_DIR/machineset-crd-multi-excluded.yaml" >/dev/null 2>&1; then
            log_success "✅ Both excluded fields removed successfully"
        else
            log_error "❌ Failed to remove multiple excluded fields"
            return 1
        fi
    else
        log_warning "yq not available - skipping multiple exclusions test"
    fi
}

print_summary() {
    section_header "Test Summary"

    echo ""
    echo -e "${BLUE}Total Tests:${NC}"
    echo -e "  ${GREEN}Passed: $TESTS_PASSED${NC}"
    echo -e "  ${RED}Failed: $TESTS_FAILED${NC}"
    echo ""

    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${GREEN}✅ All tests PASSED! excludedFields feature is working correctly.${NC}"
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        return 0
    else
        echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${RED}❌ Some tests FAILED. Please review the output above.${NC}"
        echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        return 1
    fi
}

###############################################################################
# Main Execution
###############################################################################

main() {
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║  TC-04: ExcludedFields Feature Test (PR #467)                     ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    test_prerequisites || exit 1
    backup_original_crd || exit 1
    create_compatibility_requirement || exit 1
    verify_cr_status || exit 1
    verify_excluded_fields || exit 1
    test_remove_excluded_field || exit 1
    verify_field_removed || exit 1
    test_remove_non_excluded_field || true  # Continue even if this fails
    test_multiple_excluded_fields || true   # Continue even if this fails

    print_summary
}

# Run main function
main "$@"
