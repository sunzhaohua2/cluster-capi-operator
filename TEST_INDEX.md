# Test Documentation Index

> Quick navigation guide for all test-related documentation

## 📋 Available PRs

| PR | Feature | Status | Quick Start |
|----|---------|--------|-------------|
| **#459** | CRD Compatibility Checker | ✅ Testing Available | [QUICK_TEST_GUIDE.md](QUICK_TEST_GUIDE.md) |
| **#467** | excludedFields Feature | ✅ Testing Available | [TC04_QUICK_START.md](TC04_QUICK_START.md) |
| **#455** | vSphere MAPI ↔ CAPI Conversion | ✅ Testing Available | [PR455_QUICK_START.md](PR455_QUICK_START.md) |

## 🚀 I Want To...

### ...Start Testing Immediately

#### PR #459 (CRD Compatibility Checker)
→ **[QUICK_TEST_GUIDE.md](QUICK_TEST_GUIDE.md)** - One-page quick reference
```bash
./test-manifests/comprehensive-test.sh
```

#### PR #467 (excludedFields)
→ **[TC04_QUICK_START.md](TC04_QUICK_START.md)** - 5-minute quick start
```bash
./test-manifests/tc04-excluded-fields-test.sh
```

#### PR #455 (vSphere Conversion)
→ **[PR455_QUICK_START.md](PR455_QUICK_START.md)** - vSphere testing guide
```bash
make test-conversion-vsphere
```

### ...Understand What Each PR Does

#### PR #459: CRD Compatibility Checker

→ **[TESTING_README.md](TESTING_README.md)** - Complete overview

Key Points:
- Implements Webhook validation to block destructive CRD changes
- Protects OpenShift controllers from breaking schema changes
- **Note:** CRD finalizer feature not yet implemented

#### PR #467: excludedFields Feature

→ **[TC03_VS_TC04_COMPARISON.md](TC03_VS_TC04_COMPARISON.md)** - Comparison with TC-03 (中文)

Key Points:
- Allows specific fields to be excluded from compatibility checks
- Enables graceful field deprecation
- Extension of PR #459 functionality

#### PR #455: vSphere MAPI ↔ CAPI Conversion

→ **[PR455_QUICK_START.md](PR455_QUICK_START.md)** - Quick start guide

Key Points:
- Enables bidirectional conversion for vSphere resources
- Supports vSphere-specific v1beta2 conditions
- Includes MAPI → CAPI and CAPI → MAPI conversion

### ...Run the Core Test (TC-03: Field Removal Protection)

→ **[tc03-manual-test.sh](test-manifests/tc03-manual-test.sh)** - Automated script
```bash
./test-manifests/tc03-manual-test.sh
```

→ **[MANUAL_TEST_TC03_REMOVE_FIELD.md](MANUAL_TEST_TC03_REMOVE_FIELD.md)** - Step-by-step manual instructions

### ...Test Excluded Fields Feature (TC-04: PR #467)

→ **[tc04-excluded-fields-test.sh](test-manifests/tc04-excluded-fields-test.sh)** - Automated script
```bash
./test-manifests/tc04-excluded-fields-test.sh
```

→ **[MANUAL_TEST_TC04_EXCLUDED_FIELDS.md](MANUAL_TEST_TC04_EXCLUDED_FIELDS.md)** - Step-by-step manual instructions

→ **[TC03_VS_TC04_COMPARISON.md](TC03_VS_TC04_COMPARISON.md)** - Comparison guide (中文)

### ...Test vSphere MAPI ↔ CAPI Conversion (PR #455)

→ **[PR455_QUICK_START.md](PR455_QUICK_START.md)** - Quick start guide
```bash
make test-conversion-vsphere
```

→ **[MANUAL_TEST_PR455_VSPHERE_CONVERSION.md](MANUAL_TEST_PR455_VSPHERE_CONVERSION.md)** - Step-by-step manual instructions

### ...Run All Tests

→ **[comprehensive-test.sh](test-manifests/comprehensive-test.sh)** - Full test suite
```bash
./test-manifests/comprehensive-test.sh
```

→ **[TEST_PLAN_PR459.md](TEST_PLAN_PR459.md)** - Complete test plan with all 6 test cases

### ...Test with Existing CRDs

→ **[use-existing-crd.sh](test-manifests/use-existing-crd.sh)** - Use real cluster CRDs
```bash
./test-manifests/use-existing-crd.sh machinesets.cluster.x-k8s.io
```

### ...Understand the Field Removal Test

→ **[TEST_SCENARIO_FIELD_REMOVAL.md](test-manifests/TEST_SCENARIO_FIELD_REMOVAL.md)** - Complete scenario explanation

Explains:
- Why this test is critical
- How Webhook protects against destructive changes
- Real-world scenarios

### ...Debug Finalizer Issues

→ **[diagnose-finalizer.sh](test-manifests/diagnose-finalizer.sh)** - Diagnostic tool
```bash
./test-manifests/diagnose-finalizer.sh
```

→ **[DEBUG_FINALIZER_ISSUE.md](test-manifests/DEBUG_FINALIZER_ISSUE.md)** - Debugging guide

→ **[FINALIZER_ISSUE_SUMMARY.md](FINALIZER_ISSUE_SUMMARY.md)** - Known issue summary

### ...Understand Why Finalizer Doesn't Work

→ **[BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md](BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md)** - Detailed analysis

**TL;DR:** PR #459 doesn't implement CRD finalizer functionality - it's not a bug, it's an unimplemented feature.

## 📁 File Organization

```
cluster-capi-operator/
│
├── TESTING_README.md                          ← Start here for overview (PR #459)
├── QUICK_TEST_GUIDE.md                        ← Quick reference (PR #459)
├── TEST_INDEX.md                              ← This file
│
├── TEST_PLAN_PR459.md                         ← Detailed test plan (PR #459)
├── MANUAL_TEST_TC03_REMOVE_FIELD.md          ← Manual TC-03 steps (PR #459)
├── MANUAL_TEST_TC04_EXCLUDED_FIELDS.md       ← Manual TC-04 steps (PR #467)
├── TC03_VS_TC04_COMPARISON.md                ← TC-03 vs TC-04 comparison (中文)
├── TC04_QUICK_START.md                        ← TC-04 quick start (PR #467, 中文)
│
├── PR455_QUICK_START.md                       ← vSphere conversion quick start (PR #455)
├── MANUAL_TEST_PR455_VSPHERE_CONVERSION.md   ← Manual vSphere test steps (PR #455)
│
├── FINALIZER_ISSUE_SUMMARY.md                ← Known issue summary (PR #459)
├── BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md ← Detailed bug analysis (PR #459)
│
└── test-manifests/
    ├── comprehensive-test.sh                  ← Run all tests (PR #459)
    ├── tc03-manual-test.sh                   ← Run TC-03 only (field protection, PR #459)
    ├── tc04-excluded-fields-test.sh          ← Run TC-04 only (excluded fields, PR #467)
    ├── use-existing-crd.sh                   ← Test with real CRDs (PR #459)
    ├── diagnose-finalizer.sh                 ← Debug tool (PR #459)
    │
    ├── TEST_SCENARIO_FIELD_REMOVAL.md        ← Field removal scenario (PR #459)
    └── DEBUG_FINALIZER_ISSUE.md              ← Debugging guide (PR #459)
```

## 🎯 By Role

### QE/Tester

**Start Here:**

**For PR #459 (CRD Compatibility):**
1. [QUICK_TEST_GUIDE.md](QUICK_TEST_GUIDE.md) - Understand what to test
2. [comprehensive-test.sh](test-manifests/comprehensive-test.sh) - Run automated tests
3. [TEST_PLAN_PR459.md](TEST_PLAN_PR459.md) - See all test cases

**For PR #467 (excludedFields):**
1. [TC04_QUICK_START.md](TC04_QUICK_START.md) - 5-minute quick start (中文)
2. [tc04-excluded-fields-test.sh](test-manifests/tc04-excluded-fields-test.sh) - Automated test
3. [MANUAL_TEST_TC04_EXCLUDED_FIELDS.md](MANUAL_TEST_TC04_EXCLUDED_FIELDS.md) - Manual steps

**For PR #455 (vSphere Conversion):**
1. [PR455_QUICK_START.md](PR455_QUICK_START.md) - Quick start guide
2. [MANUAL_TEST_PR455_VSPHERE_CONVERSION.md](MANUAL_TEST_PR455_VSPHERE_CONVERSION.md) - Manual steps
3. Run unit tests: `make test-conversion-vsphere`

### Developer

**Start Here:**

**For PR #459:**
1. [TESTING_README.md](TESTING_README.md) - Technical overview
2. [BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md](BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md) - Code analysis
3. [TEST_SCENARIO_FIELD_REMOVAL.md](test-manifests/TEST_SCENARIO_FIELD_REMOVAL.md) - Core functionality

**For PR #467:**
1. [TC03_VS_TC04_COMPARISON.md](TC03_VS_TC04_COMPARISON.md) - Feature comparison (中文)
2. Review `vendor/github.com/openshift/api/apiextensions/v1alpha1/types_compatibilityrequirement.go`

**For PR #455:**
1. [PR455_QUICK_START.md](PR455_QUICK_START.md) - Overview and architecture
2. Review `pkg/conversion/mapi2capi/vsphere.go` and `pkg/conversion/capi2mapi/vsphere.go`
3. Review `pkg/util/conditions.go` for v1beta2 condition handling

### PR Reviewer

**Start Here:**

**For PR #459:**
1. [TESTING_README.md](TESTING_README.md) - What's implemented
2. [TEST_PLAN_PR459.md](TEST_PLAN_PR459.md) - Test coverage
3. [FINALIZER_ISSUE_SUMMARY.md](FINALIZER_ISSUE_SUMMARY.md) - Known limitations

**For PR #467:**
1. [TC03_VS_TC04_COMPARISON.md](TC03_VS_TC04_COMPARISON.md) - How it extends PR #459 (中文)
2. [MANUAL_TEST_TC04_EXCLUDED_FIELDS.md](MANUAL_TEST_TC04_EXCLUDED_FIELDS.md) - Validation steps

**For PR #455:**
1. [PR455_QUICK_START.md](PR455_QUICK_START.md) - Feature overview
2. [MANUAL_TEST_PR455_VSPHERE_CONVERSION.md](MANUAL_TEST_PR455_VSPHERE_CONVERSION.md) - Complete test plan
3. Check unit test coverage in `pkg/conversion/*/vsphere_test.go`

## 📊 Test Case Quick Reference

| ID | Test | PR | Script | Documentation |
|----|------|-------|--------|---------------|
| TC-01 | Create matching CR | #459 | [comprehensive-test.sh](test-manifests/comprehensive-test.sh) | [TEST_PLAN_PR459.md](TEST_PLAN_PR459.md#tc-01) |
| TC-02 | Invalid CRD status | #459 | [comprehensive-test.sh](test-manifests/comprehensive-test.sh) | [TEST_PLAN_PR459.md](TEST_PLAN_PR459.md#tc-02) |
| **TC-03** | **Field removal protection** ⭐ | **#459** | [tc03-manual-test.sh](test-manifests/tc03-manual-test.sh) | [MANUAL_TEST_TC03_REMOVE_FIELD.md](MANUAL_TEST_TC03_REMOVE_FIELD.md) |
| **TC-04** | **Excluded fields** ⭐ | **#467** | [tc04-excluded-fields-test.sh](test-manifests/tc04-excluded-fields-test.sh) | [MANUAL_TEST_TC04_EXCLUDED_FIELDS.md](MANUAL_TEST_TC04_EXCLUDED_FIELDS.md) |
| TC-05 | Field addition | #459 | [comprehensive-test.sh](test-manifests/comprehensive-test.sh) | [TEST_PLAN_PR459.md](TEST_PLAN_PR459.md#tc-05) |
| TC-06 | Invalid schema | #459 | [comprehensive-test.sh](test-manifests/comprehensive-test.sh) | [TEST_PLAN_PR459.md](TEST_PLAN_PR459.md#tc-06) |
| TC-07 | Webhook failsafe | #459 | [comprehensive-test.sh](test-manifests/comprehensive-test.sh) | [TEST_PLAN_PR459.md](TEST_PLAN_PR459.md#tc-07) |

**对比说明：**
- **TC-03** (PR #459): 删除字段 → ❌ 拒绝
- **TC-04** (PR #467): 删除排除的字段 → ✅ 允许
- 详细对比参见：[TC03_VS_TC04_COMPARISON.md](TC03_VS_TC04_COMPARISON.md)

## ⚡ Quick Commands

### PR #459 (CRD Compatibility Checker)

```bash
# Run all tests
cd test-manifests && ./comprehensive-test.sh

# Run TC-03 (field removal protection)
cd test-manifests && ./tc03-manual-test.sh

# Quick test with existing CRD
cd test-manifests && ./use-existing-crd.sh machinesets.cluster.x-k8s.io

# Diagnose issues
cd test-manifests && ./diagnose-finalizer.sh

# Check Webhook status
oc get validatingwebhookconfiguration | grep compatibility

# Check Operator logs
oc logs -n openshift-compatibility-requirements-operator -l app=compatibility-requirements-controllers
```

### PR #467 (excludedFields)

```bash
# Run TC-04 (excluded fields)
cd test-manifests && ./tc04-excluded-fields-test.sh

# Run both TC-03 and TC-04 (complete validation)
cd test-manifests && ./tc03-manual-test.sh && ./tc04-excluded-fields-test.sh

# Manual test steps (中文)
cat TC04_QUICK_START.md
```

### PR #455 (vSphere Conversion)

```bash
# Run vSphere conversion unit tests
make test-conversion-vsphere

# Or run specific tests
go test -v ./pkg/conversion/mapi2capi -run TestVSphere
go test -v ./pkg/conversion/capi2mapi -run TestVSphere

# Check vSphere provider registered
oc logs -n openshift-cluster-api -l app=cluster-capi-operator | grep -i vsphere

# List vSphere resources
oc get vspheremachinetemplates -n openshift-cluster-api
oc get machines.cluster.x-k8s.io -n openshift-cluster-api

# Quick validation
cat PR455_QUICK_START.md
```

## 🔍 Decision Tree

```
Need to test PR #459?
│
├─ First time? → QUICK_TEST_GUIDE.md
│                 └─ Run: comprehensive-test.sh
│
├─ TC-03 only? → tc03-manual-test.sh
│                 └─ Fails? → diagnose-finalizer.sh
│
├─ Full validation? → TEST_PLAN_PR459.md
│                      └─ Run: comprehensive-test.sh
│
├─ Manual testing? → MANUAL_TEST_TC03_REMOVE_FIELD.md
│                     └─ Step by step guide
│
└─ Debugging? → DEBUG_FINALIZER_ISSUE.md
                 └─ Finalizer not working? → FINALIZER_ISSUE_SUMMARY.md
```

## 📝 Document Types

### 🎯 Action-Oriented (Do This)
- [QUICK_TEST_GUIDE.md](QUICK_TEST_GUIDE.md)
- [tc03-manual-test.sh](test-manifests/tc03-manual-test.sh)
- [comprehensive-test.sh](test-manifests/comprehensive-test.sh)
- [diagnose-finalizer.sh](test-manifests/diagnose-finalizer.sh)

### 📚 Explanation (Understand This)
- [TESTING_README.md](TESTING_README.md)
- [TEST_SCENARIO_FIELD_REMOVAL.md](test-manifests/TEST_SCENARIO_FIELD_REMOVAL.md)
- [FINALIZER_ISSUE_SUMMARY.md](FINALIZER_ISSUE_SUMMARY.md)

### 📋 Reference (Look This Up)
- [TEST_PLAN_PR459.md](TEST_PLAN_PR459.md)
- [MANUAL_TEST_TC03_REMOVE_FIELD.md](MANUAL_TEST_TC03_REMOVE_FIELD.md)
- [DEBUG_FINALIZER_ISSUE.md](test-manifests/DEBUG_FINALIZER_ISSUE.md)

### 🐛 Analysis (Deep Dive)
- [BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md](BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md)

## 🔗 External Links

### Pull Requests

- **PR #459**: https://github.com/openshift/cluster-capi-operator/pull/459 (CRD Compatibility Checker)
- **PR #467**: https://github.com/openshift/cluster-capi-operator/pull/467 (excludedFields Feature)
- **PR #455**: https://github.com/openshift/cluster-capi-operator/pull/455 (vSphere MAPI ↔ CAPI Conversion)

### Design Documents

- **CRD Compatibility Checker Enhancement**: https://github.com/openshift/enhancements/blob/master/enhancements/cluster-api/crd-compatibility-checker.md
- **Cluster API vSphere Provider**: https://github.com/kubernetes-sigs/cluster-api-provider-vsphere

---

**Not sure where to start?** → [TESTING_README.md](TESTING_README.md)

**Want to test now?** → `./test-manifests/comprehensive-test.sh` 🚀
