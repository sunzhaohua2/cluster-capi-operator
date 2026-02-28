# PR #455 Quick Start Guide - vSphere MAPI ↔ CAPI Conversion

## One-Line Summary

**PR #455 enables bidirectional conversion between vSphere Machine API and Cluster API resources, including v1beta2 condition handling for vSphere.**

## Prerequisites (60 seconds)

```bash
# 1. Verify you have a vSphere cluster
export KUBECONFIG=/path/to/vsphere/kubeconfig
oc get infrastructure cluster -o jsonpath='{.status.platformStatus.type}'
# Must output: VSphere

# 2. Confirm Machines exist
oc get machines -n openshift-machine-api

# 3. Checkout PR #455
cd /Users/zhsun/go/src/github.com/openshift/cluster-capi-operator
git fetch origin pull/455/head:pr-455
git checkout pr-455
```

## 5-Minute Quick Test

### Option 1: Automated Test (Unit Tests)

```bash
# Run all vSphere conversion tests
make test-conversion-vsphere

# Or run specific tests
go test -v ./pkg/conversion/mapi2capi -run TestVSphere
go test -v ./pkg/conversion/capi2mapi -run TestVSphere
```

**Expected Output:**
```
✅ All vSphere conversion tests PASSED
```

### Option 2: Manual Cluster Test

**If you have a live vSphere cluster with PR #455 deployed:**

```bash
# 1. Pick a worker machine
MACHINE_NAME=$(oc get machines -n openshift-machine-api \
  -l machine.openshift.io/cluster-api-machine-role=worker \
  -o jsonpath='{.items[0].metadata.name}')

echo "Testing with: $MACHINE_NAME"

# 2. Check if CAPI Machine exists (conversion happened)
CAPI_MACHINE=$(oc get machines.cluster.x-k8s.io -n openshift-cluster-api \
  --selector=machine.openshift.io/mapi-machine-name=$MACHINE_NAME \
  -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

if [ -n "$CAPI_MACHINE" ]; then
  echo "✅ MAPI → CAPI conversion working! CAPI Machine: $CAPI_MACHINE"

  # 3. Check vSphere template
  TEMPLATE=$(oc get machine $CAPI_MACHINE -n openshift-cluster-api \
    -o jsonpath='{.spec.infrastructureRef.name}')
  echo "✅ vSphereMachineTemplate: $TEMPLATE"

  # 4. Verify v1beta2 conditions (vSphere-specific)
  V1BETA2_CONDITIONS=$(oc get machine $CAPI_MACHINE -n openshift-cluster-api \
    -o jsonpath='{.status.v1beta2.conditions}')
  if [ -n "$V1BETA2_CONDITIONS" ]; then
    echo "✅ v1beta2 conditions found (vSphere-specific)"
  fi
else
  echo "⚠️  CAPI Machine not found. Check if migration is enabled."
fi
```

**Expected Output:**
```
Testing with: vsphere-worker-0-abc123
✅ MAPI → CAPI conversion working! CAPI Machine: vsphere-worker-0-abc123-capi
✅ vSphereMachineTemplate: vsphere-worker-0-abc123-template
✅ v1beta2 conditions found (vSphere-specific)
```

## Core Test Cases

| Test | What It Validates | Time |
|------|------------------|------|
| **TC-01** | vSphere provider registered | 1 min |
| **TC-02** | MAPI → CAPI Machine conversion | 2 min |
| **TC-03** | CAPI → MAPI Machine conversion | 2 min |
| **TC-04** | Round-trip conversion (data integrity) | 3 min |
| **TC-05** | v1beta2 conditions handling | 2 min |
| **TC-06** | MachineSet MAPI → CAPI conversion | 2 min |
| **TC-07** | MachineSet scaling sync | 3 min |
| **TC-08** | Fuzz tests (unit test) | 5 min |

**Total manual testing time: ~20 minutes**

## Quick Validation

### 1. Check vSphere Provider Registered

```bash
oc logs -n openshift-cluster-api -l app=cluster-capi-operator \
  --tail=100 | grep -i "vsphere.*registered"
```

**✅ Expected:** `Registered vSphere MAPI to CAPI conversion handler`

### 2. Verify vSphereMachineTemplate Created

```bash
oc get vspheremachinetemplates -n openshift-cluster-api

# Check a template's content
oc get vspheremachinetemplates -n openshift-cluster-api -o yaml | head -50
```

**✅ Expected:** Templates exist with vSphere configuration (datacenter, template, network)

### 3. Test Round-Trip Conversion

```bash
# Get MAPI Machine vSphere config
MACHINE_NAME=<your-machine-name>
oc get machine $MACHINE_NAME -n openshift-machine-api \
  -o jsonpath='{.spec.providerSpec.value.template}' && echo

# Get CAPI vSphereMachineTemplate config
CAPI_MACHINE=<corresponding-capi-machine>
TEMPLATE=$(oc get machine $CAPI_MACHINE -n openshift-cluster-api \
  -o jsonpath='{.spec.infrastructureRef.name}')
oc get vspheremachinetemplates $TEMPLATE -n openshift-cluster-api \
  -o jsonpath='{.spec.template.spec.template}' && echo

# Should match!
```

**✅ Expected:** Template names match between MAPI and CAPI

## Key Differences from Other Providers

### vSphere vs AWS/Azure/GCP

| Feature | AWS/Azure/GCP | vSphere (PR #455) |
|---------|---------------|-------------------|
| **Conditions Location** | `status.conditions` | `status.v1beta2.conditions` |
| **Conversion Functions** | Existing | **NEW in PR #455** |
| **Infrastructure Type** | AWSMachine, AzureMachine | **vSphereMachine** |
| **Condition Utilities** | `GetCondition()` | **`GetV1Beta2Condition()`** |

### v1beta2 Conditions Explained

vSphere uses a different condition structure:

```yaml
# AWS/Azure/GCP (v1beta1 conditions)
status:
  conditions:
    - type: Ready
      status: "True"

# vSphere (v1beta2 conditions)
status:
  v1beta2:
    conditions:
      - type: Ready
        status: "True"
```

**Why this matters:**
- PR #455 adds `GetV1Beta2Condition()` and `GetConditionStatusFromInfraObject()` utilities
- These functions check BOTH v1beta1 and v1beta2 locations
- Ensures backward compatibility while supporting vSphere

## Common Issues

### Issue 1: "vSphere conversion handler not found"

**Check:**
```bash
# Verify you're using PR #455 branch
git branch --show-current
# Should show: pr-455 or vsphere-conversion

# Check if vSphere files exist
ls -la pkg/conversion/mapi2capi/vsphere.go
ls -la pkg/conversion/capi2mapi/vsphere.go
```

### Issue 2: CAPI Machines not created

**Check:**
```bash
# Is the cluster vSphere?
oc get infrastructure cluster -o jsonpath='{.status.platformStatus.type}'

# Are controllers running?
oc get pods -n openshift-cluster-api

# Check controller logs
oc logs -n openshift-cluster-api -l app=cluster-capi-operator --tail=50
```

### Issue 3: v1beta2 conditions not found

**Check:**
```bash
# Some CAPI versions may not have v1beta2 yet
oc get crd machines.cluster.x-k8s.io -o yaml | grep -A 10 "v1beta2"

# Check both v1beta1 and v1beta2 locations
oc get machine <capi-machine> -n openshift-cluster-api \
  -o jsonpath='{.status}' | jq .
```

## Files Added/Modified in PR #455

```
pkg/conversion/
├── mapi2capi/
│   ├── vsphere.go                 ← NEW: MAPI → CAPI conversion
│   ├── vsphere_test.go           ← NEW: Unit tests
│   └── vsphere_fuzz_test.go      ← NEW: Fuzz tests
├── capi2mapi/
│   ├── vsphere.go                 ← NEW: CAPI → MAPI conversion
│   ├── vsphere_test.go           ← NEW: Unit tests
│   └── vsphere_fuzz_test.go      ← NEW: Fuzz tests
pkg/util/
└── conditions.go                  ← MODIFIED: Added v1beta2 support
pkg/controllers/
├── machinesync/                   ← MODIFIED: vSphere handler registered
└── machinesetsync/                ← MODIFIED: vSphere handler registered
cmd/machine-api-migration/
└── main.go                        ← MODIFIED: vSphere enabled
```

## Comparison with PR #459 and PR #467

| PR | Feature | Test Focus |
|----|---------|-----------|
| **#459** | CRD Compatibility Checker | Webhook validation, field protection |
| **#467** | excludedFields | Allow specific field removal |
| **#455** | vSphere Conversion | MAPI ↔ CAPI bidirectional conversion |

**PR #455 is INDEPENDENT** - It's about conversion, not validation.

## Next Steps

### For QE/Testing:
1. **Unit Tests**: Run `make test-conversion-vsphere`
2. **Manual Tests**: Follow [MANUAL_TEST_PR455_VSPHERE_CONVERSION.md](MANUAL_TEST_PR455_VSPHERE_CONVERSION.md)
3. **E2E Tests**: Deploy to vSphere cluster and verify conversion

### For Development:
1. **Review Code**: Check `pkg/conversion/*/vsphere.go`
2. **Understand v1beta2**: See `pkg/util/conditions.go`
3. **Add Tests**: Extend `vsphere_test.go` if needed

## More Information

- **Full Manual Test Guide**: [MANUAL_TEST_PR455_VSPHERE_CONVERSION.md](MANUAL_TEST_PR455_VSPHERE_CONVERSION.md)
- **PR #455**: https://github.com/openshift/cluster-capi-operator/pull/455
- **vSphere CAPI Docs**: https://github.com/kubernetes-sigs/cluster-api-provider-vsphere

## Quick Command Reference

```bash
# Build and test (unit tests)
make test-conversion-vsphere

# Check vSphere provider registered
oc logs -n openshift-cluster-api -l app=cluster-capi-operator | grep -i vsphere

# List vSphere resources
oc get vspheremachinetemplates -n openshift-cluster-api
oc get machines.cluster.x-k8s.io -n openshift-cluster-api

# Compare MAPI and CAPI Machine counts
echo "MAPI: $(oc get machines -n openshift-machine-api | wc -l)"
echo "CAPI: $(oc get machines.cluster.x-k8s.io -n openshift-cluster-api | wc -l)"
```

---

**Ready to test?** → Start with [MANUAL_TEST_PR455_VSPHERE_CONVERSION.md](MANUAL_TEST_PR455_VSPHERE_CONVERSION.md) 🚀
