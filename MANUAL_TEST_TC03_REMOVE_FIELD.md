# Manual Test Steps: TC-03 Field Removal Validation

## 📋 Test Objective

Verify that the Webhook can **block destructive CRD field removal operations**.

This is the most critical functional test for PR #459!

## ⚠️ Important Safety Notice

- ✅ This test uses `--dry-run=server` mode and will **NOT** actually modify the CRD
- ✅ All modifications are performed in temporary files
- ✅ Even if errors occur, the cluster will not be damaged
- ⚠️ If not using dry-run, test only in non-production environments

## 🎯 Test Scenario

**Background Story**:
1. OpenShift CAPI operator depends on the `machinesets.cluster.x-k8s.io` CRD
2. OpenShift controller needs the `spec.replicas` field to manage machine counts
3. If an external component (like HyperShift) upgrades the CRD and removes the `spec.replicas` field
4. OpenShift controller will stop working (destructive change)
5. **The Webhook MUST block this type of modification**

## 📝 Prerequisites Check

### 1. Check Cluster Connection

```bash
oc whoami
# Should display your username
```

### 2. Check Operator Running Status

```bash
# Check operator namespace
oc get namespace openshift-compatibility-requirements-operator

# Check operator pod
oc get pods -n openshift-compatibility-requirements-operator

# Expected output:
# NAME                                                   READY   STATUS    RESTARTS   AGE
# compatibility-requirements-controllers-xxxxx-xxxxx     1/1     Running   0          1h
```

### 3. Check Target CRD Exists

```bash
oc get crd machinesets.cluster.x-k8s.io

# Expected output:
# NAME                                 CREATED AT
# machinesets.cluster.x-k8s.io         2026-02-25T00:00:00Z
```

### 4. Check Webhook Configuration

```bash
oc get validatingwebhookconfiguration | grep compatibility

# Expected output should include:
# openshift-compatibility-requirements-apiextensions-k8s-io-v1-customresourcedefinition-validation
```

If all checks pass, you can proceed with testing.

---

## 🧪 Test Steps

### Step 1: Create Working Directory

```bash
mkdir -p /tmp/tc03-test
cd /tmp/tc03-test

echo "Working directory: $(pwd)"
```

### Step 2: Export Target CRD's Complete Schema

```bash
echo "=== Exporting MachineSet CRD ==="

oc get crd machinesets.cluster.x-k8s.io -o yaml | \
  yq eval 'del(.status, .metadata.creationTimestamp, .metadata.generation, .metadata.resourceVersion, .metadata.uid, .metadata.managedFields, .metadata.annotations)' - \
  > machineset-crd-original.yaml

echo "✓ CRD exported to: machineset-crd-original.yaml"
ls -lh machineset-crd-original.yaml
```

**Verification**: File size should be approximately 10-50KB

### Step 3: Check if replicas Field Exists

```bash
echo "=== Checking spec.replicas field ==="

# Find replicas field definition
grep -A 5 "replicas:" machineset-crd-original.yaml | head -10

# Or use yq to view
yq eval '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas' \
  machineset-crd-original.yaml
```

**Expected Output** (example):
```yaml
description: Replicas is the number of desired replicas
format: int32
type: integer
```

If you see the above output, the replicas field exists.

### Step 4: Create CompatibilityRequirement

```bash
echo "=== Creating CompatibilityRequirement ==="

# Prepare embedded CRD YAML (add indentation)
CRD_YAML_INDENTED=$(cat machineset-crd-original.yaml | sed 's/^/        /')

# Create CompatibilityRequirement manifest
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

echo "✓ CompatibilityRequirement manifest created"
```

### Step 5: Apply CompatibilityRequirement

```bash
echo "=== Applying CompatibilityRequirement ==="

oc apply -f compatibility-requirement-tc03.yaml

echo "✓ CompatibilityRequirement created"
```

### Step 6: Wait and Verify Status

```bash
echo "=== Waiting for reconciliation ==="
echo "Waiting 10 seconds..."
sleep 10

echo "=== Checking Status ==="
oc get compatibilityrequirement tc03-machineset-protection \
  -o jsonpath='{.status.conditions}' | jq

# Extract key status
ADMITTED=$(oc get compatibilityrequirement tc03-machineset-protection \
  -o jsonpath='{.status.conditions[?(@.type=="Admitted")].status}')
COMPATIBLE=$(oc get compatibilityrequirement tc03-machineset-protection \
  -o jsonpath='{.status.conditions[?(@.type=="Compatible")].status}')

echo ""
echo "Status Summary:"
echo "  Admitted: $ADMITTED"
echo "  Compatible: $COMPATIBLE"

if [ "$ADMITTED" = "True" ] && [ "$COMPATIBLE" = "True" ]; then
    echo "✓ Status correct - ready to continue testing"
else
    echo "✗ Status abnormal - check output above"
    exit 1
fi
```

**Expected Output**:
```json
[
  {
    "type": "Progressing",
    "status": "False",
    ...
  },
  {
    "type": "Admitted",
    "status": "True",
    "message": "CompatibilityRequirement admitted"
  },
  {
    "type": "Compatible",
    "status": "True",
    "message": "CRD schema is compatible"
  }
]

Status Summary:
  Admitted: True
  Compatible: True
✓ Status correct - ready to continue testing
```

If status is incorrect, stop testing and troubleshoot.

---

### Step 7: 🎯 Create CRD Version with replicas Field Removed (Core Step)

```bash
echo "=== Creating modified CRD version (replicas field removed) ==="

# Method 1: Use yq to delete field
yq eval 'del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas)' \
  machineset-crd-original.yaml > machineset-crd-without-replicas.yaml

echo "✓ Created version with replicas field removed"
echo "  Original file: machineset-crd-original.yaml"
echo "  Modified file: machineset-crd-without-replicas.yaml"
```

### Step 8: Verify replicas Field Was Removed

```bash
echo "=== Verifying replicas field removal ==="

echo "Replicas definition in original CRD:"
yq eval '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas' \
  machineset-crd-original.yaml || echo "(field exists)"

echo ""
echo "Replicas query in modified CRD:"
yq eval '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas' \
  machineset-crd-without-replicas.yaml || echo "(field not found)"
```

**Expected Output**:
```
Replicas definition in original CRD:
description: Replicas is the number of desired replicas
format: int32
type: integer

Replicas query in modified CRD:
(field not found)
```

### Step 9: 🔥 Attempt to Apply Modification (Using dry-run)

This is the **CRITICAL TEST STEP**!

```bash
echo "========================================="
echo "    Core Test: Applying CRD with Removed Field"
echo "========================================="
echo ""
echo "Using --dry-run=server mode (safe, will not actually modify)"
echo ""

# Save output to file for analysis
oc apply -f machineset-crd-without-replicas.yaml --dry-run=server \
  2>&1 | tee apply-result.txt

# Save exit code
EXIT_CODE=$?

echo ""
echo "Exit code: $EXIT_CODE"
```



### Step 10: Analyze Results ✅

```bash
echo "========================================="
echo "           Result Analysis"
echo "========================================="

if [ $EXIT_CODE -eq 0 ]; then
    echo "❌ TEST FAILED!"
    echo ""
    echo "Modification was allowed, but should have been rejected!"
    echo ""
    echo "Possible reasons:"
    echo "  1. Webhook not working correctly"
    echo "  2. CompatibilityRequirement status not Admitted=True"
    echo "  3. Webhook configuration error"
    echo ""
    echo "Please check:"
    cat apply-result.txt
else
    echo "Checking rejection reason..."
    echo ""

    if grep -qi "denied\|rejected" apply-result.txt; then
        if grep -qi "compatibility\|webhook" apply-result.txt; then
            echo "✅ TEST PASSED!"
            echo ""
            echo "Webhook correctly rejected field removal operation"
            echo ""
            echo "Rejection message:"
            cat apply-result.txt
        else
            echo "⚠️ PARTIAL PASS"
            echo ""
            echo "Modification was rejected, but possibly not by Webhook"
            echo ""
            echo "Rejection message:"
            cat apply-result.txt
        fi
    else
        echo "⚠️ UNKNOWN ERROR"
        echo ""
        cat apply-result.txt
    fi
fi
```

---

## ✅ Expected Success Result

If the test **PASSES**, you should see:

```
=========================================
           Result Analysis
=========================================
Checking rejection reason...

✅ TEST PASSED!

Webhook correctly rejected field removal operation

Rejection message:
Error from server: admission webhook "compatibilityrequirement.apiextensions.openshift.io" denied the request:
removing field 'spec.replicas' from version 'v1beta1' would break compatibility requirement tc03-machineset-protection
```

Key indicators:
- ✅ `denied the request` - Request was denied
- ✅ `compatibility requirement` - Rejected by compatibility checker
- ✅ `removing field` - Explicitly identifies field removal issue

---

## ❌ Expected Failure Scenarios

### Scenario 1: Webhook Not Working

If you see:
```
✗ TEST FAILED!
Modification was allowed, but should have been rejected!
```

**Troubleshooting Steps**:

1. Check CompatibilityRequirement status
   ```bash
   oc get compatibilityrequirement tc03-machineset-protection -o yaml
   ```
   Confirm `Admitted: True`

2. Check Webhook pod
   ```bash
   oc get pods -n openshift-compatibility-requirements-operator
   ```

3. Check Webhook logs
   ```bash
   OPERATOR_POD=$(oc get pods -n openshift-compatibility-requirements-operator \
     -o jsonpath='{.items[0].metadata.name}')
   oc logs -n openshift-compatibility-requirements-operator $OPERATOR_POD --tail=50
   ```

4. Check ValidatingWebhookConfiguration
   ```bash
   oc get validatingwebhookconfiguration \
     openshift-compatibility-requirements-apiextensions-k8s-io-v1-customresourcedefinition-validation \
     -o yaml
   ```

### Scenario 2: Rejected by Other Admission Controller

If you see rejection but not from Webhook:
```
⚠️ PARTIAL PASS
Modification was rejected, but possibly not by Webhook
```

Check error message, it might be:
- RBAC permission issue
- CRD format error
- Other admission controller

---

## 🧹 Cleanup Test Resources

After testing, cleanup:

```bash
echo "=== Cleaning up test resources ==="

# Delete CompatibilityRequirement
oc delete compatibilityrequirement tc03-machineset-protection

echo "✓ CompatibilityRequirement deleted"

# Clean temporary files (optional)
cd ~
rm -rf /tmp/tc03-test

echo "✓ Temporary files cleaned"
```

---

## 📊 Complete Test Script (One-Click Version)

Integrate all steps into a single script:

```bash
#!/bin/bash
# TC-03 Complete Test Script

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}=== TC-03: Field Removal Test ===${NC}"
echo ""

# 1. Create working directory
mkdir -p /tmp/tc03-test
cd /tmp/tc03-test

# 2. Export CRD
echo "Exporting CRD..."
oc get crd machinesets.cluster.x-k8s.io -o yaml | \
  yq eval 'del(.status, .metadata.creationTimestamp, .metadata.generation, .metadata.resourceVersion, .metadata.uid, .metadata.managedFields, .metadata.annotations)' - \
  > machineset-crd-original.yaml

# 3. Create CompatibilityRequirement
echo "Creating CompatibilityRequirement..."
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

oc apply -f compatibility-requirement-tc03.yaml

# 4. Wait for status
echo "Waiting for reconciliation..."
sleep 10

ADMITTED=$(oc get compatibilityrequirement tc03-machineset-protection \
  -o jsonpath='{.status.conditions[?(@.type=="Admitted")].status}')

if [ "$ADMITTED" != "True" ]; then
    echo -e "${RED}✗ Status incorrect, skipping test${NC}"
    exit 1
fi

echo -e "${GREEN}✓ CompatibilityRequirement admitted${NC}"

# 5. Remove replicas field
echo "Creating CRD version with replicas removed..."
yq eval 'del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas)' \
  machineset-crd-original.yaml > machineset-crd-without-replicas.yaml

# 6. Attempt to apply (critical test)
echo ""
echo -e "${YELLOW}=== Core Test: Applying CRD with Removed Field ===${NC}"
echo ""

if oc apply -f machineset-crd-without-replicas.yaml --dry-run=server 2>&1 | tee apply-result.txt; then
    echo ""
    echo -e "${RED}❌ TEST FAILED! Modification was allowed${NC}"
    exit 1
else
    if grep -qi "compatibility\|webhook" apply-result.txt; then
        echo ""
        echo -e "${GREEN}✅ TEST PASSED! Webhook correctly rejected field removal${NC}"
        echo ""
        cat apply-result.txt
    else
        echo ""
        echo -e "${YELLOW}⚠️ Modification rejected, but possibly not by Webhook${NC}"
        cat apply-result.txt
    fi
fi

# 7. Cleanup
echo ""
echo "Cleaning up..."
oc delete compatibilityrequirement tc03-machineset-protection

echo -e "${GREEN}✓ Test complete${NC}"
```

Save as `tc03-manual-test.sh`, then execute:

```bash
chmod +x tc03-manual-test.sh
./tc03-manual-test.sh
```

---

## 🎓 Understanding Test Principles

### Why Test replicas Removal?

**replicas is a critical field**:
- MachineSet uses replicas to control machine replica count
- OpenShift controller depends on this field
- Removing it would break the controller

### Webhook Workflow

```
1. User/System attempts to modify machinesets.cluster.x-k8s.io CRD
          ↓
2. Kubernetes API Server intercepts request
          ↓
3. Calls ValidatingWebhook: compatibility-requirements-controller
          ↓
4. Webhook retrieves existing CompatibilityRequirement
          ↓
5. Compares new CRD schema vs schema in CompatibilityRequirement
          ↓
6. Detects replicas field removal
          ↓
7. Returns Denied + error message
          ↓
8. API Server rejects modification request
```

---

## 📞 Troubleshooting

### Q: CompatibilityRequirement creation fails

**A**: Check YAML format
```bash
yq eval . compatibility-requirement-tc03.yaml
# Should output with no errors
```

### Q: Status never becomes Admitted=True

**A**: Check controller logs
```bash
OPERATOR_POD=$(oc get pods -n openshift-compatibility-requirements-operator \
  -o jsonpath='{.items[0].metadata.name}')
oc logs -n openshift-compatibility-requirements-operator $OPERATOR_POD
```

### Q: dry-run reports other errors

**A**: Ensure exported CRD is valid
```bash
oc apply -f machineset-crd-original.yaml --dry-run=server
# Original version should pass
```

---

## 🔗 Related Documentation

- [Comprehensive Test Script](test-manifests/comprehensive-test.sh) - Automated version
- [TEST_PLAN_PR459.md](TEST_PLAN_PR459.md) - Complete test plan
- [QUICK_TEST_GUIDE.md](QUICK_TEST_GUIDE.md) - Quick reference

---

**Test Focus**: When you see the Webhook rejecting field removal requests, TC-03 passes! This proves PR #459's core functionality works correctly.
