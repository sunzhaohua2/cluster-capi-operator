# Test Scenario: Field Removal Validation

## Overview

This is the most critical test scenario for PR #459, verifying that the Webhook can block destructive CRD field removal operations.

## 🎯 Test Objectives

Verify that when a CompatibilityRequirement exists, the Webhook can:
1. ✅ Detect when a field is removed from CRD schema
2. ✅ Reject this destructive modification
3. ✅ Return a clear error message

## 📚 Background

### Problem Scenario

In actual production environments:

1. **OpenShift installs CAPI components**
   - Installs cluster-api-operator
   - Depends on `machinesets.cluster.x-k8s.io` CRD
   - OpenShift controller uses fields like `spec.replicas`

2. **User deploys HyperShift**
   - HyperShift also manages CAPI CRDs
   - May use different CAPI versions

3. **HyperShift upgrades**
   - Upgrades to new CAPI version
   - New version may remove or rename fields
   - Attempts to update cluster CRDs

4. **Problem occurs**
   - If CRD's `spec.replicas` is removed
   - OpenShift controller cannot read this field
   - **Results in OpenShift functionality failure**

### Solution

Webhook validation implemented in PR #459:

```
OpenShift creates CompatibilityRequirement
    ↓
Records CRD schema required by OpenShift
    ↓
HyperShift attempts to update CRD (remove field)
    ↓
Webhook intercepts and compares new vs old schema
    ↓
Detects field removal
    ↓
Rejects modification + returns error message
    ↓
Protects OpenShift controller from impact
```

## 🧪 Test Methods

### Method 1: Automated Script (Recommended)

```bash
cd /Users/zhsun/go/src/github.com/openshift/cluster-capi-operator/test-manifests

# Run TC-03 test
./tc03-manual-test.sh
```

**Advantages**:
- ✅ Fully automated, one-click execution
- ✅ Includes all validation steps
- ✅ Clear Pass/Fail determination
- ✅ Automatic resource cleanup

**Output Example** (success):
```
=========================================
Core Test: Applying CRD with Removed Field
=========================================

Executing: oc apply -f machineset-crd-without-replicas.yaml --dry-run=server

Error from server: admission webhook "..." denied the request:
removing field 'spec.replicas' would break compatibility

=========================================
           ✅ TEST PASSED!
=========================================

✓ Webhook correctly rejected field removal operation
```

### Method 2: Detailed Manual Steps

Refer to detailed documentation: [MANUAL_TEST_TC03_REMOVE_FIELD.md](../MANUAL_TEST_TC03_REMOVE_FIELD.md)

Includes:
- 10 detailed steps
- Commands and expected output for each step
- Complete verification process
- Troubleshooting guide

### Method 3: Comprehensive Test Suite

Run complete tests including TC-03:

```bash
./comprehensive-test.sh
```

This runs all 6 test cases, including TC-03.

## 🔑 Key Verification Points

### Expected Success Result

When test **PASSES**, you should see:

1. **Webhook rejects request**
   ```
   Error from server: admission webhook "..." denied the request
   ```

2. **Explicitly identifies compatibility issue**
   ```
   removing field 'spec.replicas' from version 'v1beta1'
   would break compatibility requirement tc03-machineset-protection
   ```

3. **CRD remains unmodified**
   ```bash
   oc get crd machinesets.cluster.x-k8s.io -o yaml | grep replicas
   # Should still find replicas field
   ```

### Expected Failure Scenarios

If test **FAILS**, you might see:

1. **Modification allowed** (serious issue!)
   ```
   customresourcedefinition.apiextensions.k8s.io/machinesets.cluster.x-k8s.io configured (dry run)
   ```
   This indicates Webhook is not working.

2. **Rejected for other reasons**
   ```
   Error: ... RBAC ...
   Error: ... invalid format ...
   ```
   This indicates it wasn't rejected by Webhook, needs investigation.

## 🔍 Test Variants

### Variant 1: Test Different Fields

Besides `spec.replicas`, you can test removing other fields:

```bash
# Remove spec.selector
yq eval 'del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.selector)' \
  original.yaml > without-selector.yaml

# Remove spec.template
yq eval 'del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.template)' \
  original.yaml > without-template.yaml
```

All field removal operations should be rejected.

### Variant 2: Test Allowed Modifications

Contrast test to verify Webhook doesn't incorrectly reject legitimate changes:

```bash
# Add new field (should be allowed)
yq eval '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.newField = {"type": "string"}' \
  original.yaml > with-new-field.yaml

oc apply -f with-new-field.yaml --dry-run=server
# Expected: Success
```

### Variant 3: Test Warn Mode

Modify CompatibilityRequirement to Warn mode:

```yaml
spec:
  compatibilitySchema:
    customResourceDefinitionSchemaValidation:
      action: Warn  # Change to Warn
```

Expected:
- Modification is allowed
- But returns warning message

## 📊 Test Matrix

| Operation | CompatibilityRequirement | Expected Result |
|-----------|-------------------------|-----------------|
| Remove required field | action: Deny | ❌ Rejected |
| Remove optional field | action: Deny | ❌ Rejected |
| Add new field | action: Deny | ✅ Allowed |
| Modify description | action: Deny | ✅ Allowed |
| Remove field | action: Warn | ⚠️ Warning but allowed |
| Remove field | No CR | ✅ Allowed |

## 🐛 Common Issues

### Q1: Test shows modification was allowed

**Troubleshooting steps**:

1. Check CompatibilityRequirement status
   ```bash
   oc get compatibilityrequirement tc03-machineset-protection \
     -o jsonpath='{.status.conditions}' | jq
   ```
   Confirm `Admitted: True`

2. Check Webhook configuration
   ```bash
   oc get validatingwebhookconfiguration | grep compatibility
   ```

3. View operator logs
   ```bash
   OPERATOR_POD=$(oc get pods -n openshift-compatibility-requirements-operator \
     -o jsonpath='{.items[0].metadata.name}')
   oc logs -n openshift-compatibility-requirements-operator $OPERATOR_POD --tail=50
   ```

### Q2: replicas field doesn't exist

Some CRD versions may not have the replicas field.

**Solution**: Choose another field to test, or use a different CRD:

```bash
# Use machines.cluster.x-k8s.io
# Remove spec.providerID field
```

### Q3: Other admission errors appear

If you see RBAC or other errors, it's not a Webhook issue.

**Solution**: Ensure:
- Using a user with appropriate permissions
- CRD format is correct
- Using original exported CRD (only removing one field)

## 📈 Test Coverage

This test scenario covers:

- ✅ Webhook registration and invocation
- ✅ CompatibilityRequirement parsing
- ✅ Schema comparison logic
- ✅ Field removal detection
- ✅ Admission denial mechanism
- ✅ Error message return

**Not covered** (requires other tests):
- ❌ CRD Finalizer (not implemented)
- ❌ ExcludedFields feature (not implemented)
- ❌ Object validation (not implemented)

## 🔗 Related Resources

### Documentation
- [Detailed Manual Test Steps](../MANUAL_TEST_TC03_REMOVE_FIELD.md)
- [Complete Test Plan](../TEST_PLAN_PR459.md)
- [Quick Test Guide](../QUICK_TEST_GUIDE.md)

### Scripts
- [tc03-manual-test.sh](tc03-manual-test.sh) - TC-03 automation script
- [comprehensive-test.sh](comprehensive-test.sh) - Complete test suite
- [diagnose-finalizer.sh](diagnose-finalizer.sh) - Diagnostic tool

### PR and EP
- PR: https://github.com/openshift/cluster-capi-operator/pull/459
- EP: crd-compatibility-checker.md

## 🎓 Technical Details

### Webhook Implementation Location

```
cluster-capi-operator/
└── pkg/
    └── controllers/
        └── crdcompatibility/
            ├── crdvalidation/
            │   └── crdvalidator_webhook.go  ← Webhook validation logic
            ├── controller.go
            └── reconcile.go
```

### Key Code Snippet

Webhook field removal detection logic (pseudocode):

```go
func ValidateCRDUpdate(newCRD, oldCRD *CRD, compatReq *CompatibilityRequirement) error {
    // Get expected schema from CompatibilityRequirement
    expectedSchema := compatReq.Spec.CompatibilitySchema.Data

    // Compare new CRD against expected schema
    removedFields := compareSchemas(expectedSchema, newCRD.Spec.Schema)

    if len(removedFields) > 0 {
        // Deny or Warn based on action setting
        if compatReq.Spec.Action == "Deny" {
            return fmt.Errorf("removing fields %v would break compatibility", removedFields)
        }
    }

    return nil  // Allow
}
```

## 📝 Test Report Template

After testing, you can use this template to report results:

```markdown
### TC-03: Field Removal Validation - Test Report

**Test Date**: 2026-02-26
**Tester**: [Your Name]
**Cluster Version**: [oc version]
**Operator Version**: PR #459

**Test Result**: ✅ PASS / ❌ FAIL

**Detailed Output**:
```
[Paste test script output]
```

**Verification Points**:
- [ ] Webhook rejected field removal request
- [ ] Error message includes "compatibility"
- [ ] Error message explicitly identifies which field was removed
- [ ] CRD was not modified

**Issues and Observations**:
[Any problems found or special circumstances]
```

---

**Quick Start**: `./tc03-manual-test.sh` 🚀
