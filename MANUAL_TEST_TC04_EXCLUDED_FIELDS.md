# Manual Test TC-04: Excluded Fields Feature

## Overview

This test validates PR #467 functionality: `excludedFields` allows excluding specific fields from compatibility checks. When a field is excluded, removing it from the CRD will NOT be rejected by the admission webhook.

**Comparison with TC-03:**
- **TC-03**: Remove a required field → Admission **DENIED** ❌
- **TC-04**: Remove a field listed in `excludedFields` → Admission **ALLOWED** ✅

## Prerequisites

- OpenShift cluster with cluster-capi-operator PR #467 deployed
- CRD Compatibility Checker webhook running
- `oc` CLI configured and authenticated
- Cluster must have MachineSet CRD installed

## Test Objective

Verify that when a field path is specified in `excludedFields`, the admission webhook allows CRD updates that remove that field.

## Test Steps

### Step 1: Export Current MachineSet CRD

```bash
export KUBECONFIG=/path/to/your/kubeconfig

# Export the current MachineSet CRD
oc get crd machinesets.cluster.x-k8s.io -o yaml > /tmp/machineset-crd-original.yaml

# Verify it has the replicas field
oc get crd machinesets.cluster.x-k8s.io -o jsonpath='{.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas}' | jq .
```

**Expected Output:**
```json
{
  "default": 1,
  "description": "Replicas is the number of desired replicas...",
  "format": "int32",
  "type": "integer"
}
```

### Step 2: Create CompatibilityRequirement with excludedFields

Create a CompatibilityRequirement that **excludes** the `spec.replicas` field from validation:

```bash
# Get the current CRD and prepare it for the CompatibilityRequirement
oc get crd machinesets.cluster.x-k8s.io -o yaml | \
  sed 's/^/        /' > /tmp/crd-indented.yaml

# Get the storage version
STORAGE_VERSION=$(oc get crd machinesets.cluster.x-k8s.io -o jsonpath='{.spec.versions[?(@.storage==true)].name}')
echo "Storage version: $STORAGE_VERSION"

# Create the CompatibilityRequirement with excludedFields
cat <<EOF | oc apply -f -
apiVersion: apiextensions.openshift.io/v1alpha1
kind: CompatibilityRequirement
metadata:
  name: tc04-machineset-excluded-fields
spec:
  compatibilitySchema:
    customResourceDefinition:
      name: machinesets.cluster.x-k8s.io
      type: YAML
      data: |
$(cat /tmp/crd-indented.yaml)
    requiredVersions:
      defaultSelection: StorageOnly
    excludedFields:
      - path: "spec.replicas"
        versions:
          - $STORAGE_VERSION
  customResourceDefinitionSchemaValidation:
    action: Deny
EOF
```

**Expected Output:**
```
compatibilityrequirement.apiextensions.openshift.io/tc04-machineset-excluded-fields created
```

### Step 3: Verify CompatibilityRequirement Status

Wait for the CompatibilityRequirement to be processed:

```bash
# Check status conditions
oc get compatibilityrequirement tc04-machineset-excluded-fields -o jsonpath='{.status.conditions[?(@.type=="Admitted")]}' | jq .

# Should show Admitted=True
oc get compatibilityrequirement tc04-machineset-excluded-fields -o jsonpath='{.status.conditions[?(@.type=="Compatible")]}' | jq .

# Should show Compatible=True
```

**Expected Output:**
```json
{
  "type": "Admitted",
  "status": "True",
  "reason": "AdmissionSucceeded",
  "message": "..."
}
```

### Step 4: Verify excludedFields Configuration

Confirm that the excludedFields are correctly configured:

```bash
oc get compatibilityrequirement tc04-machineset-excluded-fields \
  -o jsonpath='{.spec.compatibilitySchema.excludedFields}' | jq .
```

**Expected Output:**
```json
[
  {
    "path": "spec.replicas",
    "versions": [
      "v1beta1"  // or your storage version
    ]
  }
]
```

### Step 5: Check ValidatingWebhookConfiguration

Verify the webhook is configured:

```bash
oc get validatingwebhookconfiguration -l app=crd-compatibility-checker

# Check webhook details
oc get validatingwebhookconfiguration \
  crd-compatibility-checker-validating-webhook-configuration \
  -o yaml | grep -A 10 "name: vcrd.openshift.io"
```

### Step 6: Remove the Excluded Field from CRD

Now attempt to remove the `spec.replicas` field from the CRD. Since it's in `excludedFields`, this should be **ALLOWED**:

```bash
# Create a modified CRD without spec.replicas
oc get crd machinesets.cluster.x-k8s.io -o yaml > /tmp/machineset-crd-modified.yaml

# Remove spec.replicas using yq or manual editing
# If you have yq:
yq eval 'del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas)' \
  /tmp/machineset-crd-modified.yaml > /tmp/machineset-crd-no-replicas.yaml

# Or manually edit the file to remove the replicas field
# Then apply the modified CRD
oc apply -f /tmp/machineset-crd-no-replicas.yaml
```

**Expected Output:**
```
customresourcedefinition.apiextensions.k8s.io/machinesets.cluster.x-k8s.io configured
```

**✅ This should SUCCEED because spec.replicas is in excludedFields**

### Step 7: Verify Field Was Actually Removed

Confirm that the `replicas` field is no longer in the CRD schema:

```bash
# This should return empty or null
oc get crd machinesets.cluster.x-k8s.io \
  -o jsonpath='{.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas}'

# Check ResourceVersion changed
oc get crd machinesets.cluster.x-k8s.io -o jsonpath='{.metadata.resourceVersion}'
```

**Expected Output:**
```
# Empty output (field removed)

# ResourceVersion should be different from original
105210 → 112500 (example)
```

### Step 8: Verify Webhook Allowed the Change

Check the webhook logs to confirm it processed and allowed the change:

```bash
# Get webhook pod name
WEBHOOK_POD=$(oc get pods -n openshift-cluster-api \
  -l app=crd-compatibility-checker \
  --field-selector=status.phase=Running \
  -o jsonpath='{.items[0].metadata.name}')

# Check logs
oc logs -n openshift-cluster-api $WEBHOOK_POD --tail=50 | \
  grep -A 5 -B 5 "machinesets.cluster.x-k8s.io"
```

**Expected Log Messages:**
```
"Validating CRD update for machinesets.cluster.x-k8s.io"
"Excluded field spec.replicas from validation"
"CRD validation succeeded: compatible with requirements"
"Admission allowed: true"
```

### Step 9: Test Removing a Non-Excluded Field (Should Fail)

Now test that removing a field **NOT** in excludedFields is still rejected:

```bash
# Try to remove spec.selector (not excluded)
# This should be DENIED

# Create another modified CRD without spec.selector
yq eval 'del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.selector)' \
  /tmp/machineset-crd-original.yaml > /tmp/machineset-crd-no-selector.yaml

oc apply -f /tmp/machineset-crd-no-selector.yaml
```

**Expected Output:**
```
Error from server: admission webhook "vcrd.openshift.io" denied the request:
CRD update violates compatibility requirement "tc04-machineset-excluded-fields":
field spec.selector is required but missing in the updated CRD
```

**✅ This should FAIL because spec.selector is NOT in excludedFields**

### Step 10: Test Multiple Excluded Fields

Create a CompatibilityRequirement with multiple excluded fields:

```bash
cat <<EOF | oc apply -f -
apiVersion: apiextensions.openshift.io/v1alpha1
kind: CompatibilityRequirement
metadata:
  name: tc04-machineset-multiple-excluded
spec:
  compatibilitySchema:
    customResourceDefinition:
      name: machinesets.cluster.x-k8s.io
      type: YAML
      data: |
$(cat /tmp/crd-indented.yaml)
    requiredVersions:
      defaultSelection: StorageOnly
    excludedFields:
      - path: "spec.replicas"
      - path: "spec.minReadySeconds"
      - path: "status.conditions.lastTransitionTime"
  customResourceDefinitionSchemaValidation:
    action: Deny
EOF
```

Verify both fields can be removed:

```bash
# Remove both replicas and minReadySeconds
yq eval 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.replicas) |
         del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.minReadySeconds)' \
  /tmp/machineset-crd-original.yaml > /tmp/machineset-crd-multi-excluded.yaml

oc apply -f /tmp/machineset-crd-multi-excluded.yaml
```

**Expected:** Should succeed ✅

## Cleanup

```bash
# Delete test CompatibilityRequirements
oc delete compatibilityrequirement tc04-machineset-excluded-fields
oc delete compatibilityrequirement tc04-machineset-multiple-excluded

# Restore original CRD
oc apply -f /tmp/machineset-crd-original.yaml

# Clean up temp files
rm -f /tmp/machineset-crd-*.yaml /tmp/crd-indented.yaml
```

## Success Criteria

| Step | Test Case | Expected Result | Actual Result |
|------|-----------|----------------|---------------|
| 6 | Remove excluded field (spec.replicas) | ✅ Allowed | |
| 7 | Verify field removed | ✅ Field not present | |
| 9 | Remove non-excluded field (spec.selector) | ❌ Denied | |
| 10 | Remove multiple excluded fields | ✅ Allowed | |

## Key Differences from TC-03

| Aspect | TC-03 | TC-04 |
|--------|-------|-------|
| **excludedFields** | Not specified | `spec.replicas` excluded |
| **Remove spec.replicas** | ❌ Denied | ✅ Allowed |
| **Webhook behavior** | Strict validation | Flexible validation |
| **Use case** | Enforce compatibility | Allow field deprecation |

## Validation Points

1. ✅ **excludedFields configuration**: CompatibilityRequirement accepts excludedFields array
2. ✅ **Field exclusion works**: Removing excluded field is allowed
3. ✅ **Non-excluded fields protected**: Removing non-excluded field is still denied
4. ✅ **Multiple exclusions**: Multiple fields can be excluded simultaneously
5. ✅ **Version-specific exclusions**: Can exclude fields from specific API versions
6. ✅ **Status reporting**: CompatibilityRequirement status accurately reflects validation

## Expected Behavior

- **Excluded field removal**: When a field is in `excludedFields`, admission webhook allows CRD updates that remove the field
- **Non-excluded field removal**: Fields not in `excludedFields` are still protected and cannot be removed
- **Validation granularity**: exclusions can be version-specific using the `versions` array
- **Status conditions**: CompatibilityRequirement should still show `Compatible=True` when comparing against CRD with excluded fields removed

## Troubleshooting

### Issue: Field removal still denied despite being in excludedFields

**Check:**
```bash
# Verify excludedFields syntax
oc get compatibilityrequirement tc04-machineset-excluded-fields -o yaml | yq .spec.compatibilitySchema.excludedFields

# Verify path format (should be dot-separated, no leading dot)
# ✅ Correct: "spec.replicas"
# ❌ Wrong: ".spec.replicas", "spec/replicas"

# Verify version matches
oc get crd machinesets.cluster.x-k8s.io -o jsonpath='{.spec.versions[?(@.storage==true)].name}'
```

### Issue: CompatibilityRequirement not admitted

**Check:**
```bash
oc get compatibilityrequirement tc04-machineset-excluded-fields -o jsonpath='{.status.conditions}' | jq .

# Look for validation errors in message field
```

## Notes

- **Field path format**: Paths are dot-separated (e.g., `spec.replicas`, `status.conditions.type`)
- **Array fields**: For array fields like `status.conditions`, the path applies to all items
- **Nested fields**: Can exclude deeply nested fields (e.g., `spec.template.spec.containers.image`)
- **Version scoping**: If `versions` is omitted, the field is excluded from all versions
- **Maximum exclusions**: Up to 64 fields can be excluded per CompatibilityRequirement

## Related Documentation

- PR #467: https://github.com/openshift/cluster-capi-operator/pull/467
- Enhancement Proposal: Section on field deprecation
- API Reference: `APIExcludedField` type definition
