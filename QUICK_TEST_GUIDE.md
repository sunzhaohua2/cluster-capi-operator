# 🚀 Quick Test Guide - PR #459

## One Minute Quick Start

```bash
# 1. Ensure you're logged into the cluster
oc whoami

# 2. Run tests
cd /Users/zhsun/go/src/github.com/openshift/cluster-capi-operator/test-manifests
chmod +x comprehensive-test.sh
./comprehensive-test.sh
```

## 📊 What Are We Testing?

PR #459 implements **Webhook Validation** functionality to prevent destructive CRD modifications.

### ✅ Core Features (4 Validation Points)

1. **Create CompatibilityRequirement**
   - Reference existing CRD schema
   - Verify CR creation succeeds

2. **Observe Status**
   - `Admitted: True` - CR is accepted
   - `Compatible: True` - Schema is compatible

3. **Try to Remove a Field** ⭐⭐⭐ Most Important
   - Simulate removing a field from the CRD
   - **Should be rejected by Webhook**

4. **Verify Protection Works**
   - Webhook blocks destructive modifications
   - Protects OpenShift controller

## 🎯 Focus Test: TC-03

**This is the most important test!**

```bash
# Manual test steps
# 1. Create CompatibilityRequirement
oc apply -f test-manifests/use-existing-crd.sh machinesets.cluster.x-k8s.io

# 2. Wait for status
sleep 10

# 3. Backup CRD
oc get crd machinesets.cluster.x-k8s.io -o yaml > /tmp/backup.yaml

# 4. Remove a field (test modification)
yq eval 'del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas)' \
  /tmp/backup.yaml > /tmp/modified.yaml

# 5. Attempt to apply (use dry-run!)
oc apply -f /tmp/modified.yaml --dry-run=server

# Expected result: Rejected
# Error: admission webhook denied the request: removing field would break compatibility
```

## 📝 Expected Results Summary

| Test | Expected Result | Importance |
|------|----------------|------------|
| TC-01 | Status: Admitted=True, Compatible=True | High |
| TC-02 | Invalid CRD shows error in Status | Medium |
| **TC-03** | **Field removal is rejected** | **Critical** ⭐ |
| TC-04 | Field addition is allowed | High |
| TC-05 | Invalid schema is handled | Low |
| TC-06 | Webhook failurePolicy=Fail | High |

## ⚠️ Known Issues

### CRD Finalizer Not Implemented

**Symptom**:
```bash
$ oc get crd machinesets.cluster.x-k8s.io -o jsonpath='{.metadata.finalizers}'
# Result: Empty (even though CompatibilityRequirement exists)
```

**Cause**: PR #459 code doesn't implement CRD finalizer addition

**Impact**:
- ✅ Webhook validation **works normally**
- ❌ CRD deletion protection **doesn't work**

**Details**: See [BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md](BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md)

## 📂 File Navigation

```
cluster-capi-operator/
├── TEST_PLAN_PR459.md              # Detailed test plan ⭐
├── QUICK_TEST_GUIDE.md             # This file (quick reference)
├── BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md  # Bug analysis
├── FINALIZER_ISSUE_SUMMARY.md      # Finalizer issue summary
└── test-manifests/
    ├── comprehensive-test.sh        # Automated test script ⭐⭐⭐
    ├── tc03-manual-test.sh          # TC-03 focused test ⭐⭐
    ├── use-existing-crd.sh          # Test with existing CRD
    ├── diagnose-finalizer.sh        # Finalizer diagnostic tool
    └── TEST_SCENARIO_FIELD_REMOVAL.md  # Field removal scenario
```

## 🔍 If Tests Fail

### TC-03 Fails: Field removal not rejected

**Possible reasons**:
1. Webhook not properly registered
2. Webhook pod not running
3. CompatibilityRequirement status is not Admitted=True

**Diagnosis**:
```bash
# Check Webhook configuration
oc get validatingwebhookconfiguration | grep compatibility

# Check Webhook pod
oc get pods -n openshift-compatibility-requirements-operator

# Check CompatibilityRequirement status
oc get compatibilityrequirement <name> -o jsonpath='{.status.conditions}' | jq
```

### Webhook Related Issues

```bash
# Check Webhook logs
OPERATOR_POD=$(oc get pods -n openshift-compatibility-requirements-operator \
  -o jsonpath='{.items[0].metadata.name}')
oc logs -n openshift-compatibility-requirements-operator $OPERATOR_POD | grep -i webhook

# Check failurePolicy
oc get validatingwebhookconfiguration \
  openshift-compatibility-requirements-apiextensions-k8s-io-v1-customresourcedefinition-validation \
  -o jsonpath='{.webhooks[0].failurePolicy}'
```

## 💡 Best Practices

### ✅ DO

- Use `--dry-run=server` to test CRD modifications
- Backup CRD before testing
- Wait sufficient time for Status to update (10+ seconds)
- Check Webhook pod health status

### ❌ DON'T

- ❌ Don't remove fields from real CRDs in production
- ❌ Don't skip TC-03 (most important test)
- ❌ Don't expect finalizer to work (not implemented)
- ❌ Don't test unimplemented features (ExcludedFields, etc.)

## 📞 Reporting Issues

If TC-03 fails (field removal not rejected), this is a **critical issue** and should be reported in the PR:

```markdown
## Bug: Webhook Did Not Block Field Removal

**Test**: TC-03 - Field removal should be rejected
**Result**: ❌ FAIL - Webhook allowed field removal

**Reproduction Steps**:
1. Created CompatibilityRequirement (status: Admitted=True)
2. Attempted to remove field from CRD
3. Modification was allowed (expected: should be rejected)

**Environment**:
- OpenShift version: [run oc version]
- Webhook status: [oc get validatingwebhookconfiguration]

**Logs**:
[Paste webhook pod logs]
```

## 🎓 Test Theory

### Why is TC-03 Most Important?

**Scenario**:
1. OpenShift installs CAPI operator, depends on `machinesets.cluster.x-k8s.io` CRD
2. OpenShift controller needs `spec.replicas` field
3. User deploys HyperShift, which also manages CAPI CRDs
4. HyperShift upgrades and removes `spec.replicas` field
5. **Without protection, OpenShift controller crashes**

**Webhook's Role**:
- CompatibilityRequirement records OpenShift's required schema
- Webhook intercepts CRD modifications
- If modification would remove fields OpenShift needs → Reject
- Protects OpenShift controller from damage

### Schema Compatibility Rules

**Allowed modifications** (✅):
- Add new optional fields
- Modify description or metadata
- Add new enum values

**Forbidden modifications** (❌):
- Remove existing fields
- Change field types
- Add new required fields
- Tighten validation rules

## 🔗 Related Links

- **PR**: https://github.com/openshift/cluster-capi-operator/pull/459
- **EP**: crd-compatibility-checker.md
- **Detailed Test Plan**: [TEST_PLAN_PR459.md](TEST_PLAN_PR459.md)

---

**Quick Command Reference**:

```bash
# Run all tests
./test-manifests/comprehensive-test.sh

# Run TC-03 only
./test-manifests/tc03-manual-test.sh

# Diagnose finalizer issues
./test-manifests/diagnose-finalizer.sh

# Quick test with existing CRD
./test-manifests/use-existing-crd.sh machinesets.cluster.x-k8s.io

# View CompatibilityRequirement
oc get compatibilityrequirement

# View Webhook configuration
oc get validatingwebhookconfiguration | grep compatibility
```

---

**Start Immediately**: `./test-manifests/comprehensive-test.sh` 🚀
