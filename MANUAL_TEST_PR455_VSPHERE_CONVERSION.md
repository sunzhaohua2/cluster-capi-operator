# Manual Test Guide: PR #455 - vSphere MAPI ↔ CAPI Conversion

## Overview

PR #455 (SPLAT-2410) adds vSphere support to the Machine API to Cluster API conversion framework. This test validates:

- **MAPI → CAPI conversion**: vSphere Machine API resources convert to Cluster API format
- **CAPI → MAPI conversion**: Cluster API resources convert back to Machine API format
- **Round-trip conversion**: Data integrity is maintained through bidirectional conversion
- **v1beta2 conditions**: vSphere-specific condition handling works correctly

## Prerequisites

### Cluster Requirements

- vSphere-based OpenShift cluster (4.18+)
- Cluster API operator installed (from PR #455 branch: `vsphere-conversion`)
- Access to modify Machine and MachineSet resources
- `oc` CLI configured and authenticated

### Build and Deploy PR #455

```bash
# Clone and checkout PR #455
cd /Users/zhsun/go/src/github.com/openshift/cluster-capi-operator
git fetch origin pull/455/head:pr-455
git checkout pr-455

# Or checkout the branch directly
git checkout vsphere-conversion

# Build the operator image
make docker-build IMG=<your-registry>/cluster-capi-operator:vsphere-test

# Push the image
make docker-push IMG=<your-registry>/cluster-capi-operator:vsphere-test

# Deploy to cluster
oc patch deployment cluster-capi-operator -n openshift-cluster-api \
  --type='json' \
  -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/image", "value":"<your-registry>/cluster-capi-operator:vsphere-test"}]'

# Verify deployment
oc get pods -n openshift-cluster-api -l app=cluster-capi-operator
```

### Verify vSphere Cluster

```bash
# Confirm cluster is vSphere platform
oc get infrastructure cluster -o jsonpath='{.status.platformStatus.type}'
# Expected: VSphere

# Check existing vSphere Machines
oc get machines -n openshift-machine-api -o wide

# Check existing vSphere MachineSets
oc get machinesets -n openshift-machine-api
```

## Test Cases

### TC-01: Verify vSphere Provider Registration

**Objective**: Confirm that vSphere conversion handlers are registered in the controllers.

```bash
# Check operator logs for vSphere registration
oc logs -n openshift-cluster-api \
  -l app=cluster-capi-operator \
  --tail=100 | grep -i "vsphere\|provider"

# Should see logs indicating vSphere handlers registered:
# - "Registered vSphere conversion handler"
# - "vSphere provider enabled"
```

**Expected Output:**
```
INFO Registered vSphere MAPI to CAPI conversion handler
INFO Registered vSphere CAPI to MAPI conversion handler
```

**Success Criteria:**
- ✅ vSphere conversion handlers registered
- ✅ No errors in operator startup logs

---

### TC-02: MAPI to CAPI Conversion for vSphere Machine

**Objective**: Test conversion from vSphere Machine API Machine to Cluster API Machine.

#### Step 1: Export Existing vSphere Machine

```bash
# Get a worker machine (not control plane)
MACHINE_NAME=$(oc get machines -n openshift-machine-api \
  -l machine.openshift.io/cluster-api-machine-role=worker \
  -o jsonpath='{.items[0].metadata.name}' | head -1)

echo "Testing with Machine: $MACHINE_NAME"

# Export the Machine
oc get machine $MACHINE_NAME -n openshift-machine-api -o yaml > /tmp/vsphere-mapi-machine.yaml

# Examine the vSphere providerSpec
oc get machine $MACHINE_NAME -n openshift-machine-api \
  -o jsonpath='{.spec.providerSpec.value}' | jq .
```

#### Step 2: Enable CAPI Migration (if not already enabled)

```bash
# Check if migration is enabled
oc get configmap cluster-capi-operator-config -n openshift-cluster-api

# If migration not enabled, you may need to enable it
# (This step depends on your cluster setup and operator configuration)
```

#### Step 3: Trigger MAPI to CAPI Conversion

The conversion happens automatically when:
1. A MAPI Machine exists
2. The machinesync controller processes it
3. A corresponding CAPI Machine should be created

```bash
# Watch for CAPI Machine creation
oc get machines.cluster.x-k8s.io -n openshift-cluster-api -w

# In another terminal, check controller logs
oc logs -n openshift-cluster-api \
  -l app=cluster-capi-operator \
  -f | grep -i "$MACHINE_NAME"
```

#### Step 4: Verify CAPI Machine Created

```bash
# Check if CAPI Machine was created
CAPI_MACHINE=$(oc get machines.cluster.x-k8s.io -n openshift-cluster-api \
  -l cluster.x-k8s.io/cluster-name=<cluster-name> \
  -o jsonpath='{.items[?(@.metadata.annotations.machine\.openshift\.io/mapi-machine-name=="'$MACHINE_NAME'")].metadata.name}')

if [ -n "$CAPI_MACHINE" ]; then
  echo "✅ CAPI Machine created: $CAPI_MACHINE"

  # Export CAPI Machine for comparison
  oc get machine $CAPI_MACHINE -n openshift-cluster-api -o yaml > /tmp/vsphere-capi-machine.yaml

  # Check vSphere-specific fields in CAPI Machine
  oc get machine $CAPI_MACHINE -n openshift-cluster-api \
    -o jsonpath='{.spec.bootstrap.dataSecretName}' && echo
else
  echo "❌ CAPI Machine NOT created"
fi
```

#### Step 5: Validate vSphere-Specific Conversion

```bash
# Compare key vSphere fields
echo "=== MAPI Machine vSphere Config ==="
oc get machine $MACHINE_NAME -n openshift-machine-api \
  -o jsonpath='{.spec.providerSpec.value}' | \
  jq '{
    workspace: .workspace,
    template: .template,
    numCPUs: .numCPUs,
    memoryMiB: .memoryMiB,
    diskGiB: .diskGiB,
    network: .network
  }'

echo "=== CAPI vSphereMachineTemplate Config ==="
# Find the associated vSphereMachineTemplate
TEMPLATE_NAME=$(oc get machine $CAPI_MACHINE -n openshift-cluster-api \
  -o jsonpath='{.spec.infrastructureRef.name}')

if [ -n "$TEMPLATE_NAME" ]; then
  oc get vspheremachinetemplates $TEMPLATE_NAME -n openshift-cluster-api \
    -o jsonpath='{.spec.template.spec}' | jq .
fi
```

**Expected Conversion Mapping:**

| MAPI Field | CAPI Field |
|------------|------------|
| `spec.providerSpec.value.workspace.datacenter` | `spec.template.spec.datacenter` |
| `spec.providerSpec.value.template` | `spec.template.spec.template` |
| `spec.providerSpec.value.numCPUs` | `spec.template.spec.numCPUs` |
| `spec.providerSpec.value.memoryMiB` | `spec.template.spec.memoryMiB` |
| `spec.providerSpec.value.diskGiB` | `spec.template.spec.diskGiB` |
| `spec.providerSpec.value.network` | `spec.template.spec.network` |

**Success Criteria:**
- ✅ CAPI Machine created with matching configuration
- ✅ vSphere-specific fields correctly converted
- ✅ No errors in controller logs

---

### TC-03: CAPI to MAPI Conversion for vSphere Machine

**Objective**: Test conversion from Cluster API Machine back to Machine API format.

**Note:** This test validates that changes to a CAPI Machine are reflected back to the corresponding MAPI Machine.

#### Step 1: Modify CAPI Machine Annotation

```bash
# Add an annotation to the CAPI Machine
oc annotate machine $CAPI_MACHINE -n openshift-cluster-api \
  test-annotation="capi-to-mapi-test"

# Wait for sync
sleep 5
```

#### Step 2: Verify Annotation Synced to MAPI Machine

```bash
# Check if annotation appears on MAPI Machine
oc get machine $MACHINE_NAME -n openshift-machine-api \
  -o jsonpath='{.metadata.annotations.test-annotation}'

# Expected: capi-to-mapi-test
```

**Success Criteria:**
- ✅ Changes to CAPI Machine propagate to MAPI Machine
- ✅ No sync errors in controller logs

---

### TC-04: Round-Trip Conversion Validation

**Objective**: Ensure data integrity through MAPI → CAPI → MAPI conversion.

```bash
# Export original MAPI Machine providerSpec
oc get machine $MACHINE_NAME -n openshift-machine-api \
  -o jsonpath='{.spec.providerSpec.value}' > /tmp/original-mapi-providerspec.json

# Get the CAPI Machine vSphereMachineTemplate
TEMPLATE_NAME=$(oc get machine $CAPI_MACHINE -n openshift-cluster-api \
  -o jsonpath='{.spec.infrastructureRef.name}')

oc get vspheremachinetemplates $TEMPLATE_NAME -n openshift-cluster-api \
  -o jsonpath='{.spec.template.spec}' > /tmp/capi-vsphere-spec.json

# Re-export MAPI Machine providerSpec after round-trip
oc get machine $MACHINE_NAME -n openshift-machine-api \
  -o jsonpath='{.spec.providerSpec.value}' > /tmp/roundtrip-mapi-providerspec.json

# Compare (some fields may differ due to defaults/omitempty)
diff -u /tmp/original-mapi-providerspec.json /tmp/roundtrip-mapi-providerspec.json
```

**Success Criteria:**
- ✅ Essential vSphere configuration fields match
- ✅ No data loss in critical fields:
  - Datacenter
  - Template
  - CPU/Memory/Disk specifications
  - Network configuration

---

### TC-05: v1beta2 Conditions Handling

**Objective**: Validate vSphere-specific v1beta2 condition handling.

vSphere uses `status.v1beta2.conditions` instead of `status.conditions` for some condition types.

#### Step 1: Check for v1beta2 Conditions in CAPI Machine

```bash
# Check if the CAPI Machine has v1beta2 conditions
oc get machine $CAPI_MACHINE -n openshift-cluster-api \
  -o jsonpath='{.status.v1beta2.conditions}' | jq .

# Expected: Array of condition objects
```

#### Step 2: Verify Condition Conversion Utilities

Check that the code correctly handles v1beta2 conditions:

```bash
# Look for v1beta2 condition handling in logs
oc logs -n openshift-cluster-api \
  -l app=cluster-capi-operator \
  --tail=200 | grep -i "v1beta2\|condition"
```

**Expected Condition Types for vSphere:**
- `Ready`
- `MachineHealthCheckSucceeded`
- `MachineOwnerRemediatedCondition`

**Success Criteria:**
- ✅ v1beta2 conditions present in CAPI vSphere machines
- ✅ Conditions correctly read using `GetV1Beta2Condition()`
- ✅ Paused state detection works with v1beta2 conditions

---

### TC-06: MachineSet MAPI to CAPI Conversion

**Objective**: Test conversion for vSphere MachineSets.

#### Step 1: Export Existing vSphere MachineSet

```bash
# Get a worker MachineSet
MACHINESET_NAME=$(oc get machinesets -n openshift-machine-api \
  -o jsonpath='{.items[0].metadata.name}')

echo "Testing with MachineSet: $MACHINESET_NAME"

# Export the MachineSet
oc get machineset $MACHINESET_NAME -n openshift-machine-api -o yaml \
  > /tmp/vsphere-mapi-machineset.yaml

# Check vSphere providerSpec
oc get machineset $MACHINESET_NAME -n openshift-machine-api \
  -o jsonpath='{.spec.template.spec.providerSpec.value}' | jq .
```

#### Step 2: Verify CAPI MachineSet Created

```bash
# Check for corresponding CAPI MachineSet
CAPI_MACHINESET=$(oc get machinesets.cluster.x-k8s.io -n openshift-cluster-api \
  -l cluster.x-k8s.io/cluster-name=<cluster-name> \
  -o jsonpath='{.items[?(@.metadata.annotations.machineset\.openshift\.io/mapi-machineset-name=="'$MACHINESET_NAME'")].metadata.name}')

if [ -n "$CAPI_MACHINESET" ]; then
  echo "✅ CAPI MachineSet created: $CAPI_MACHINESET"

  # Export CAPI MachineSet
  oc get machineset $CAPI_MACHINESET -n openshift-cluster-api -o yaml \
    > /tmp/vsphere-capi-machineset.yaml
else
  echo "❌ CAPI MachineSet NOT created"
fi
```

#### Step 3: Compare Replicas and Configuration

```bash
# Compare replica counts
echo "MAPI MachineSet replicas:"
oc get machineset $MACHINESET_NAME -n openshift-machine-api \
  -o jsonpath='{.spec.replicas}' && echo

echo "CAPI MachineSet replicas:"
oc get machineset $CAPI_MACHINESET -n openshift-cluster-api \
  -o jsonpath='{.spec.replicas}' && echo

# Compare vSphere template configuration
echo "=== Checking vSphereMachineTemplate Reference ==="
VSPHERE_TEMPLATE=$(oc get machineset $CAPI_MACHINESET -n openshift-cluster-api \
  -o jsonpath='{.spec.template.spec.infrastructureRef.name}')

echo "vSphereMachineTemplate: $VSPHERE_TEMPLATE"

oc get vspheremachinetemplates $VSPHERE_TEMPLATE -n openshift-cluster-api -o yaml
```

**Success Criteria:**
- ✅ CAPI MachineSet created with matching replica count
- ✅ vSphereMachineTemplate correctly referenced
- ✅ Template contains equivalent vSphere configuration

---

### TC-07: Scale MachineSet and Verify Sync

**Objective**: Test that scaling operations sync correctly between MAPI and CAPI.

```bash
# Get current replica count
ORIGINAL_REPLICAS=$(oc get machineset $MACHINESET_NAME -n openshift-machine-api \
  -o jsonpath='{.spec.replicas}')

echo "Original replicas: $ORIGINAL_REPLICAS"

# Scale up MAPI MachineSet
NEW_REPLICAS=$((ORIGINAL_REPLICAS + 1))
oc scale machineset $MACHINESET_NAME -n openshift-machine-api \
  --replicas=$NEW_REPLICAS

# Wait for sync
sleep 10

# Check CAPI MachineSet replicas
CAPI_REPLICAS=$(oc get machineset $CAPI_MACHINESET -n openshift-cluster-api \
  -o jsonpath='{.spec.replicas}')

echo "CAPI MachineSet replicas after scale: $CAPI_REPLICAS"

if [ "$CAPI_REPLICAS" -eq "$NEW_REPLICAS" ]; then
  echo "✅ Scaling synced correctly"
else
  echo "❌ Scaling NOT synced (expected $NEW_REPLICAS, got $CAPI_REPLICAS)"
fi

# Restore original replica count
oc scale machineset $MACHINESET_NAME -n openshift-machine-api \
  --replicas=$ORIGINAL_REPLICAS
```

**Success Criteria:**
- ✅ Replica count changes sync from MAPI to CAPI
- ✅ New Machines created in both MAPI and CAPI format
- ✅ No sync errors in controller logs

---

### TC-08: Fuzz Test Validation (Unit Test)

**Objective**: Run fuzz tests to validate conversion stability.

```bash
# Checkout PR #455 branch
git checkout vsphere-conversion

# Run vSphere fuzz tests
cd /Users/zhsun/go/src/github.com/openshift/cluster-capi-operator

# MAPI to CAPI fuzz test
go test -v ./pkg/conversion/mapi2capi -run TestFuzzVSphere -fuzz=FuzzVSphere -fuzztime=30s

# CAPI to MAPI fuzz test
go test -v ./pkg/conversion/capi2mapi -run TestFuzzVSphere -fuzz=FuzzVSphere -fuzztime=30s
```

**Success Criteria:**
- ✅ Fuzz tests complete without panics
- ✅ No conversion errors found
- ✅ All generated inputs convert successfully

---

## Verification Checklist

After completing all test cases, verify the following:

### Controller Status

```bash
# Check controller health
oc get pods -n openshift-cluster-api
oc logs -n openshift-cluster-api -l app=cluster-capi-operator --tail=50

# Check for errors
oc logs -n openshift-cluster-api -l app=cluster-capi-operator \
  --tail=200 | grep -i "error\|failed"
```

### Resource Sync Status

```bash
# Count MAPI Machines
MAPI_COUNT=$(oc get machines -n openshift-machine-api -o json | \
  jq '[.items[] | select(.status.phase != "Deleting")] | length')

# Count CAPI Machines (excluding control plane)
CAPI_COUNT=$(oc get machines.cluster.x-k8s.io -n openshift-cluster-api -o json | \
  jq '[.items[] | select(.metadata.ownerReferences[].kind != "KubeadmControlPlane")] | length')

echo "MAPI Machines: $MAPI_COUNT"
echo "CAPI Machines: $CAPI_COUNT"

# They should match (approximately, accounting for creation/deletion timing)
```

### vSphere-Specific Validation

```bash
# Verify vSphere infrastructure references
oc get vspheremachinetemplates -n openshift-cluster-api
oc get vsphereclustertemplates -n openshift-cluster-api

# Check that all vSphere templates are valid
oc get vspheremachinetemplates -n openshift-cluster-api -o json | \
  jq '.items[] | {name: .metadata.name, datacenter: .spec.template.spec.datacenter, template: .spec.template.spec.template}'
```

## Cleanup

```bash
# Remove test annotations
oc annotate machine $CAPI_MACHINE -n openshift-cluster-api \
  test-annotation-

# Restore original replica counts (if changed)
# oc scale machineset $MACHINESET_NAME -n openshift-machine-api --replicas=<original>

# Clean up temporary files
rm -f /tmp/vsphere-*.yaml /tmp/*-mapi-*.json /tmp/capi-*.json
```

## Success Criteria Summary

| Test Case | Description | Status |
|-----------|-------------|--------|
| TC-01 | vSphere provider registration | ⬜ |
| TC-02 | MAPI to CAPI Machine conversion | ⬜ |
| TC-03 | CAPI to MAPI Machine conversion | ⬜ |
| TC-04 | Round-trip conversion validation | ⬜ |
| TC-05 | v1beta2 conditions handling | ⬜ |
| TC-06 | MachineSet MAPI to CAPI conversion | ⬜ |
| TC-07 | MachineSet scaling sync | ⬜ |
| TC-08 | Fuzz test validation | ⬜ |

## Troubleshooting

### Issue: CAPI Machines Not Created

**Check:**
```bash
# Verify controller is running
oc get pods -n openshift-cluster-api -l app=cluster-capi-operator

# Check controller logs for errors
oc logs -n openshift-cluster-api -l app=cluster-capi-operator --tail=100

# Verify vSphere provider is enabled
oc logs -n openshift-cluster-api -l app=cluster-capi-operator | grep -i vsphere
```

### Issue: Conversion Errors in Logs

**Check:**
```bash
# Look for specific conversion errors
oc logs -n openshift-cluster-api -l app=cluster-capi-operator | \
  grep -A 10 "conversion error"

# Check for invalid vSphere configuration
oc get machine $MACHINE_NAME -n openshift-machine-api \
  -o jsonpath='{.spec.providerSpec.value}' | jq .
```

### Issue: v1beta2 Conditions Not Found

**Check:**
```bash
# Verify CAPI version supports v1beta2 conditions
oc get machine $CAPI_MACHINE -n openshift-cluster-api \
  -o jsonpath='{.status}' | jq .

# Check if conditions are in v1beta1 location instead
oc get machine $CAPI_MACHINE -n openshift-cluster-api \
  -o jsonpath='{.status.conditions}' | jq .
```

## Related Documentation

- **PR #455**: https://github.com/openshift/cluster-capi-operator/pull/455
- **vSphere Provider Spec**: `pkg/conversion/mapi2capi/vsphere.go`
- **Conversion Utilities**: `pkg/util/conditions.go`
- **Unit Tests**:
  - `pkg/conversion/mapi2capi/vsphere_test.go`
  - `pkg/conversion/capi2mapi/vsphere_test.go`

## Notes

- **vSphere-Specific**: This test is ONLY for vSphere clusters. For AWS/Azure/GCP, use existing conversion tests.
- **v1beta2 Conditions**: vSphere uses `status.v1beta2.conditions` while other providers use `status.conditions`
- **Round-Trip Testing**: Critical for ensuring no data loss during bidirectional conversion
- **Controller Integration**: The machinesync and machinesetsync controllers automatically perform conversion
