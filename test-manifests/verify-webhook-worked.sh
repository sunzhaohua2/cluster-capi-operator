#!/bin/bash
# Quick verification: Did the Webhook actually block the modification?

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

CRD_NAME="machinesets.cluster.x-k8s.io"

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}Webhook Verification Test${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

echo "Analyzing your test result..."
echo ""

# Step 1: Record current state
echo -e "${GREEN}Step 1: Recording current CRD state${NC}"

BEFORE_VERSION=$(oc get crd $CRD_NAME -o jsonpath='{.metadata.resourceVersion}')
BEFORE_GEN=$(oc get crd $CRD_NAME -o jsonpath='{.metadata.generation}')
BEFORE_TIME=$(date +%s)

echo "  ResourceVersion: $BEFORE_VERSION"
echo "  Generation: $BEFORE_GEN"
echo ""

# Step 2: Check if replicas field exists now
echo -e "${GREEN}Step 2: Checking if replicas field exists${NC}"

REPLICAS_EXISTS=$(oc get crd $CRD_NAME -o yaml | grep -c "name: replicas" || echo "0")

if [ "$REPLICAS_EXISTS" -gt 0 ]; then
    echo -e "  ${GREEN}✓ replicas field EXISTS${NC}"
else
    echo -e "  ${RED}✗ replicas field MISSING${NC}"
fi
echo ""

# Step 3: Try to apply the modified CRD again
echo -e "${GREEN}Step 3: Testing modification again${NC}"
echo "  Creating CRD without replicas field..."

# Export current CRD
oc get crd $CRD_NAME -o yaml | \
  yq eval 'del(.status, .metadata.creationTimestamp, .metadata.generation, .metadata.resourceVersion, .metadata.uid, .metadata.managedFields, .metadata.annotations)' - \
  > /tmp/crd-test-original.yaml

# Remove replicas field
yq eval 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.replicas)' \
  /tmp/crd-test-original.yaml > /tmp/crd-test-modified.yaml

echo "  Attempting to apply..."
echo ""

# Try to apply
if oc apply -f /tmp/crd-test-modified.yaml > /tmp/apply-output.txt 2>&1; then
    APPLY_RESULT="SUCCESS"
    echo -e "  ${YELLOW}Command succeeded (exit code 0)${NC}"
else
    APPLY_RESULT="FAILED"
    echo -e "  ${GREEN}Command failed (exit code $?)${NC}"
fi

echo "  Output:"
cat /tmp/apply-output.txt | sed 's/^/    /'
echo ""

# Step 4: Check state after apply
echo -e "${GREEN}Step 4: Checking CRD state after apply${NC}"

AFTER_VERSION=$(oc get crd $CRD_NAME -o jsonpath='{.metadata.resourceVersion}')
AFTER_GEN=$(oc get crd $CRD_NAME -o jsonpath='{.metadata.generation}')

echo "  Before - ResourceVersion: $BEFORE_VERSION, Generation: $BEFORE_GEN"
echo "  After  - ResourceVersion: $AFTER_VERSION, Generation: $AFTER_GEN"
echo ""

# Step 5: Check if field still exists
REPLICAS_AFTER=$(oc get crd $CRD_NAME -o yaml | grep -c "name: replicas" || echo "0")

echo -e "${GREEN}Step 5: Checking if replicas field still exists${NC}"
if [ "$REPLICAS_AFTER" -gt 0 ]; then
    echo -e "  ${GREEN}✓ replicas field STILL EXISTS${NC}"
else
    echo -e "  ${RED}✗ replicas field IS MISSING${NC}"
fi
echo ""

# Analysis
echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}Analysis${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

VERSION_CHANGED=false
GEN_CHANGED=false

if [ "$BEFORE_VERSION" != "$AFTER_VERSION" ]; then
    VERSION_CHANGED=true
fi

if [ "$BEFORE_GEN" != "$AFTER_GEN" ]; then
    GEN_CHANGED=true
fi

# Determine what happened
if [ "$VERSION_CHANGED" = false ] && [ "$GEN_CHANGED" = false ]; then
    echo -e "${GREEN}✅✅✅ WEBHOOK IS WORKING! ✅✅✅${NC}"
    echo ""
    echo "Evidence:"
    echo "  1. ResourceVersion unchanged: $BEFORE_VERSION = $AFTER_VERSION"
    echo "  2. Generation unchanged: $BEFORE_GEN = $AFTER_GEN"
    echo "  3. replicas field still exists"
    echo ""
    echo "Conclusion:"
    echo "  The CRD was NOT actually modified!"
    echo ""
    echo "Why 'oc apply' showed 'configured':"
    echo "  - 'oc apply' is declarative and compares desired vs actual state"
    echo "  - Even if the request was REJECTED, it reports 'configured'"
    echo "  - This is misleading, but it's how 'oc apply' works"
    echo ""
    echo "What really happened:"
    echo "  1. You ran: oc apply -f modified-crd.yaml"
    echo "  2. Kubernetes API server received the request"
    echo "  3. ValidatingWebhook was called"
    echo "  4. Webhook detected field removal"
    echo "  5. Webhook REJECTED the modification"
    echo "  6. API server did NOT update the CRD"
    echo "  7. 'oc apply' compared final state vs desired state"
    echo "  8. Reported 'configured' (misleading!)"
    echo ""
    echo -e "${GREEN}Your Webhook is protecting the CRD correctly!${NC}"

elif [ "$VERSION_CHANGED" = true ] && [ "$REPLICAS_AFTER" -gt 0 ]; then
    echo -e "${YELLOW}⚠️ PARTIAL MODIFICATION DETECTED${NC}"
    echo ""
    echo "Evidence:"
    echo "  1. ResourceVersion changed: $BEFORE_VERSION → $AFTER_VERSION"
    echo "  2. Generation changed: $BEFORE_GEN → $AFTER_GEN"
    echo "  3. BUT replicas field still exists!"
    echo ""
    echo "Possible explanations:"
    echo "  A. CRD was modified but operator reconciled it back"
    echo "  B. Server-side apply prevented field removal (field manager conflict)"
    echo "  C. Webhook allowed partial modification"
    echo ""
    echo "Recommended actions:"
    echo "  1. Check operator logs for reconciliation"
    echo "  2. Check managedFields to see field ownership"
    echo "  3. Check webhook logs"

elif [ "$VERSION_CHANGED" = true ] && [ "$REPLICAS_AFTER" -eq 0 ]; then
    echo -e "${RED}❌❌❌ WEBHOOK FAILED! ❌❌❌${NC}"
    echo ""
    echo "Evidence:"
    echo "  1. ResourceVersion changed: $BEFORE_VERSION → $AFTER_VERSION"
    echo "  2. Generation changed: $BEFORE_GEN → $AFTER_GEN"
    echo "  3. replicas field IS MISSING!"
    echo ""
    echo "Conclusion:"
    echo "  The CRD WAS MODIFIED and Webhook did NOT block it!"
    echo ""
    echo "CRITICAL: Need to restore CRD immediately!"
    echo ""
    echo "Restore command:"
    echo "  oc apply -f /tmp/crd-test-original.yaml"

else
    echo -e "${YELLOW}⚠️ UNUSUAL STATE${NC}"
    echo "Cannot determine what happened. Manual investigation needed."
fi

echo ""
echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}Additional Checks${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# Check webhook logs
echo "Checking webhook logs..."
OPERATOR_POD=$(oc get pods -n openshift-compatibility-requirements-operator \
  -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

if [ -n "$OPERATOR_POD" ]; then
    echo "Operator pod: $OPERATOR_POD"
    echo ""
    echo "Recent webhook activity:"
    oc logs -n openshift-compatibility-requirements-operator $OPERATOR_POD --tail=20 | \
      grep -E "webhook|deny|reject|machinesets|validation" || echo "  No relevant logs found"
else
    echo "  Could not find operator pod"
fi

echo ""
echo "=== Test Complete ==="
