# Testing Guide for PR #459 - CompatibilityRequirement Feature

## 📋 Overview

This directory contains comprehensive testing resources for PR https://github.com/openshift/cluster-capi-operator/pull/459, which implements CRD Compatibility Checker functionality.

**PR Status**: Implements Webhook validation and Status reporting. CRD finalizer feature is not implemented.

## 🚀 Quick Start

### Run All Tests (Recommended)

```bash
cd /Users/zhsun/go/src/github.com/openshift/cluster-capi-operator/test-manifests

# Run comprehensive test suite
./comprehensive-test.sh
```

### Run Core Test Only (TC-03)

```bash
# This is the most critical test - verifies Webhook blocks field removal
./tc03-manual-test.sh
```

## 📚 Documentation Structure

### For Quick Testing

| Document | Purpose | When to Use |
|----------|---------|-------------|
| **[QUICK_TEST_GUIDE.md](QUICK_TEST_GUIDE.md)** | One-page quick reference | First time testing, need quick overview |
| **[tc03-manual-test.sh](test-manifests/tc03-manual-test.sh)** | Automated TC-03 test | Test core Webhook functionality |

### For Comprehensive Testing

| Document | Purpose | When to Use |
|----------|---------|-------------|
| **[TEST_PLAN_PR459.md](TEST_PLAN_PR459.md)** | Complete test plan with all 6 test cases | Full PR validation |
| **[comprehensive-test.sh](test-manifests/comprehensive-test.sh)** | Automated test suite | Run all tests at once |
| **[TEST_SCENARIO_FIELD_REMOVAL.md](test-manifests/TEST_SCENARIO_FIELD_REMOVAL.md)** | Field removal scenario details | Understand TC-03 deeply |

### For Manual Testing

| Document | Purpose | When to Use |
|----------|---------|-------------|
| **[MANUAL_TEST_TC03_REMOVE_FIELD.md](MANUAL_TEST_TC03_REMOVE_FIELD.md)** | Step-by-step TC-03 manual test | Learn how to test manually |
| **[use-existing-crd.sh](test-manifests/use-existing-crd.sh)** | Quick test with existing CRD | Test with real cluster CRDs |

### For Troubleshooting

| Document | Purpose | When to Use |
|----------|---------|-------------|
| **[diagnose-finalizer.sh](test-manifests/diagnose-finalizer.sh)** | Diagnose finalizer issues | Finalizer not appearing on CRD |
| **[DEBUG_FINALIZER_ISSUE.md](test-manifests/DEBUG_FINALIZER_ISSUE.md)** | Finalizer debugging guide | Deep dive into finalizer problem |

### Known Issues

| Document | Purpose |
|----------|---------|
| **[FINALIZER_ISSUE_SUMMARY.md](FINALIZER_ISSUE_SUMMARY.md)** | Summary of finalizer not implemented |
| **[BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md](BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md)** | Detailed analysis with code examples |

## 🎯 Test Coverage

### ✅ Implemented and Tested

PR #459 implements these features, which are covered by tests:

1. **Webhook Validation** (TC-03, TC-04)
   - Detects schema changes
   - Blocks destructive modifications (field removal)
   - Allows compatible modifications (field addition)

2. **Status Reporting** (TC-01, TC-02)
   - Admitted condition
   - Compatible condition
   - Progressing condition

3. **CompatibilityRequirement Management** (TC-01, TC-05)
   - CR creation and parsing
   - Schema validation
   - Error handling

### ❌ Not Implemented (Not Tested)

These features are NOT implemented in PR #459:

- **CRD Finalizer** - Target CRDs don't get finalizers
- **ExcludedFields** - Not implemented
- **RequiredVersions.additionalVersions** - Not implemented
- **objectSchemaValidation** - Not implemented

## 🧪 Test Cases Summary

| ID | Test Case | Priority | Status |
|----|-----------|----------|--------|
| **TC-01** | Create matching CompatibilityRequirement | High | ✅ Ready |
| **TC-02** | Invalid CRD status observation | Medium | ✅ Ready |
| **TC-03** | Field removal rejected by Webhook | **Critical** | ✅ Ready ⭐ |
| **TC-04** | Field addition allowed | High | ✅ Ready |
| **TC-05** | Invalid schema handling | Low | ✅ Ready |
| **TC-06** | Webhook failurePolicy check | High | ✅ Ready |

**TC-03 is the most important test** - it validates the core purpose of PR #459.

## 📖 Testing Workflow

### For First-Time Testers

```
1. Read QUICK_TEST_GUIDE.md (5 minutes)
   ↓
2. Run tc03-manual-test.sh (2 minutes)
   ↓
3. If passes: Report success in PR
   If fails: Run diagnose-finalizer.sh
```

### For Comprehensive Validation

```
1. Read TEST_PLAN_PR459.md (15 minutes)
   ↓
2. Run comprehensive-test.sh (5 minutes)
   ↓
3. Review all test results
   ↓
4. Report findings in PR with test output
```

### For Deep Investigation

```
1. Read MANUAL_TEST_TC03_REMOVE_FIELD.md
   ↓
2. Manually execute each step
   ↓
3. Use diagnose-finalizer.sh if issues found
   ↓
4. Consult DEBUG_FINALIZER_ISSUE.md for troubleshooting
```

## 🔍 Expected Test Results

### Success Criteria

PR #459 can be considered validated when:

#### Must Pass (Critical)
- ✅ TC-01: CompatibilityRequirement creation with matching schema succeeds
- ✅ TC-03: Field removal is rejected by Webhook ⭐⭐⭐
- ✅ TC-06: Webhook configured as fail-closed

#### Should Pass (Important)
- ✅ TC-02: Invalid CRD shows correct Status
- ✅ TC-04: Field addition is allowed
- ✅ TC-05: Invalid schema is handled

#### Known Failures (Expected)
- ❌ CRD Finalizer tests - Feature not implemented

### Sample Success Output

```bash
$ ./tc03-manual-test.sh

=========================================
Core Test: Applying CRD with Removed Field
=========================================

Error from server: admission webhook "..." denied the request:
removing field 'spec.replicas' would break compatibility

=========================================
           ✅ TEST PASSED!
=========================================

✓ Webhook correctly rejected field removal operation
```

## ⚠️ Known Issues

### 1. CRD Finalizer Not Implemented

**Issue**: Target CRDs do not receive finalizers even when CompatibilityRequirement is Admitted.

**Impact**:
- ✅ Webhook validation works correctly
- ❌ CRD deletion protection doesn't work
- Users can still delete CRDs even with active CompatibilityRequirements

**Status**: Feature not implemented in PR #459

**Documentation**:
- [FINALIZER_ISSUE_SUMMARY.md](FINALIZER_ISSUE_SUMMARY.md)
- [BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md](BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md)

### 2. Test Limitations

Some tests use `--dry-run=server` mode for safety:
- Cannot verify actual CRD updates
- Only tests admission control layer
- Recommended to run in non-production for full validation

## 💡 Best Practices

### Before Testing

1. ✅ Backup important CRDs
2. ✅ Test in non-production environment first
3. ✅ Ensure Operator is running
4. ✅ Check Webhook configuration exists

### During Testing

1. ✅ Use `--dry-run=server` for CRD modifications
2. ✅ Wait sufficient time for reconciliation (10+ seconds)
3. ✅ Check Status conditions carefully
4. ✅ Review Webhook logs if tests fail

### After Testing

1. ✅ Clean up test CompatibilityRequirements
2. ✅ Verify CRDs are unchanged
3. ✅ Document any issues found
4. ✅ Report results in PR

## 📞 Reporting Results

### If Tests Pass

Comment in PR #459:

```markdown
## Test Results: PASS ✅

Ran comprehensive test suite for PR #459.

**Environment:**
- Cluster version: [oc version]
- Test date: [date]

**Results:**
- TC-01: ✅ PASS
- TC-02: ✅ PASS
- TC-03: ✅ PASS (Core validation)
- TC-04: ✅ PASS
- TC-05: ✅ PASS
- TC-06: ✅ PASS

**Core Validation (TC-03):**
Webhook successfully rejected field removal attempts with appropriate error message.

**Known Issue:**
CRD finalizer functionality is not implemented (expected).
```

### If Tests Fail

Comment in PR #459 with details:

```markdown
## Test Results: Issues Found ❌

**Failed Test:** TC-03 - Field removal not rejected

**Environment:**
- Cluster version: [oc version]
- Operator pod: [pod name and status]

**Issue:**
Webhook allowed field removal that should have been rejected.

**Reproduction:**
[Steps to reproduce]

**Logs:**
```
[Paste relevant logs]
```

**Webhook Status:**
[Output of webhook configuration check]
```

## 🔗 External References

- **PR**: https://github.com/openshift/cluster-capi-operator/pull/459
- **Enhancement Proposal**: https://github.com/openshift/enhancements/blob/master/enhancements/cluster-api/crd-compatibility-checker.md
- **Source Code**: `pkg/controllers/crdcompatibility/`

## 🛠️ Quick Command Reference

```bash
# Run all tests
./test-manifests/comprehensive-test.sh

# Run TC-03 only (most important)
./test-manifests/tc03-manual-test.sh

# Test with existing CRD
./test-manifests/use-existing-crd.sh machinesets.cluster.x-k8s.io

# Diagnose finalizer issues
./test-manifests/diagnose-finalizer.sh

# Check CompatibilityRequirements
oc get compatibilityrequirement

# Check Webhook configuration
oc get validatingwebhookconfiguration | grep compatibility

# Check Operator status
oc get pods -n openshift-compatibility-requirements-operator

# View Operator logs
OPERATOR_POD=$(oc get pods -n openshift-compatibility-requirements-operator \
  -o jsonpath='{.items[0].metadata.name}')
oc logs -n openshift-compatibility-requirements-operator $OPERATOR_POD
```

## 📝 Contributing

If you find issues with the tests or want to add new test cases:

1. Review existing test structure in `test-manifests/`
2. Follow the naming convention: `tc##-description.sh`
3. Update TEST_PLAN_PR459.md with new test cases
4. Ensure cleanup functions are included

## 🎓 Understanding the Feature

### What Problem Does This Solve?

**Scenario without protection:**
```
OpenShift installs CAPI operator → depends on CRD fields
↓
HyperShift also manages same CRDs → upgrades CAPI version
↓
New CAPI version removes fields → CRD updated
↓
OpenShift controller breaks → cluster degraded
```

**With CompatibilityRequirement + Webhook:**
```
OpenShift creates CompatibilityRequirement → records required schema
↓
HyperShift attempts CRD update → Webhook intercepts
↓
Webhook compares schemas → detects field removal
↓
Rejects update → OpenShift protected
```

### Key Concepts

- **CompatibilityRequirement**: CR that records what schema version a component needs
- **Webhook Validation**: Admission webhook that validates CRD updates
- **Schema Compatibility**: Rules defining what changes are breaking vs. safe
- **Deny vs. Warn**: Action modes for handling incompatible changes

---

**Quick Start**: `./test-manifests/comprehensive-test.sh` 🚀

**Questions?** Check [QUICK_TEST_GUIDE.md](QUICK_TEST_GUIDE.md) or [TEST_PLAN_PR459.md](TEST_PLAN_PR459.md)
