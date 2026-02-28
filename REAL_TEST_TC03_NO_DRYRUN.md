# TC-03 Real Test (Without dry-run)

## ⚠️ CRITICAL WARNINGS

**This test will ACTUALLY ATTEMPT to modify the CRD!**

- ✅ **ONLY run in non-production test clusters**
- ✅ **NEVER run in production environments**
- ✅ **Complete backup before testing**
- ✅ **Be prepared to restore if something goes wrong**

**What could go wrong:**
- If Webhook is NOT working, the CRD will be modified
- OpenShift controller may break
- Cluster may become degraded
- You MUST be able to restore the CRD

## 📋 Prerequisites

### 1. Environment Validation

```bash
# Confirm this is a TEST cluster
echo "Cluster API URL: $(oc whoami --show-server)"
echo ""
echo "⚠️ Is this a PRODUCTION cluster?"
echo "If YES, STOP NOW! Use dry-run mode instead."
echo ""
read -p "Type 'TEST-CLUSTER' to confirm this is safe to test: " CONFIRM

if [ "$CONFIRM" != "TEST-CLUSTER" ]; then
    echo "Aborting for safety"
    exit 1
fi
```

### 2. Backup Strategy

You need TWO backups:
1. **CRD backup** - To restore if modified
2. **etcd backup** - Ultimate fallback (recommended)

## 🛡️ Complete Backup Procedure

### Backup 1: Export CRD

```bash
echo "=== Creating CRD Backup ==="

BACKUP_DIR="/tmp/crd-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Backup the CRD
oc get crd machinesets.cluster.x-k8s.io -o yaml > "$BACKUP_DIR/machineset-crd-backup.yaml"

# Verify backup
if [ ! -s "$BACKUP_DIR/machineset-crd-backup.yaml" ]; then
    echo "❌ Backup failed!"
    exit 1
fi

echo "✓ CRD backed up to: $BACKUP_DIR/machineset-crd-backup.yaml"
echo "Backup size: $(wc -l < $BACKUP_DIR/machineset-crd-backup.yaml) lines"

# Save backup location
echo "$BACKUP_DIR/machineset-crd-backup.yaml" > /tmp/last-crd-backup.txt
```

### Backup 2: Snapshot Current Resource Version

```bash
echo "=== Recording CRD Resource Version ==="

ORIGINAL_VERSION=$(oc get crd machinesets.cluster.x-k8s.io \
  -o jsonpath='{.metadata.resourceVersion}')
ORIGINAL_GENERATION=$(oc get crd machinesets.cluster.x-k8s.io \
  -o jsonpath='{.metadata.generation}')

echo "Original ResourceVersion: $ORIGINAL_VERSION"
echo "Original Generation: $ORIGINAL_GENERATION"

# Save for later comparison
cat > "$BACKUP_DIR/original-metadata.txt" <<EOF
RESOURCE_VERSION=$ORIGINAL_VERSION
GENERATION=$ORIGINAL_GENERATION
BACKUP_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
EOF

echo "✓ Metadata saved"
```

### Backup 3: Verify Existing MachineSet Resources (Optional but Recommended)

```bash
echo "=== Checking Existing MachineSet Resources ==="

# List all MachineSet resources
oc get machinesets -A -o yaml > "$BACKUP_DIR/existing-machinesets.yaml"

MACHINESET_COUNT=$(oc get machinesets -A --no-headers | wc -l)
echo "Found $MACHINESET_COUNT MachineSet resources"

if [ $MACHINESET_COUNT -gt 0 ]; then
    echo "⚠️ Warning: There are existing MachineSet resources"
    echo "If CRD is broken, these resources may be affected"
fi
```

## 🧪 Real Test Procedure

### Step 1: Create CompatibilityRequirement

```bash
echo "=== Creating CompatibilityRequirement ==="

# Use the clean CRD as baseline
CRD_YAML=$(cat "$BACKUP_DIR/machineset-crd-backup.yaml" | \
  yq eval 'del(.status, .metadata.creationTimestamp, .metadata.generation, .metadata.resourceVersion, .metadata.uid, .metadata.managedFields, .metadata.annotations)' -)

CRD_YAML_INDENTED=$(echo "$CRD_YAML" | sed 's/^/        /')

cat > "$BACKUP_DIR/compatibility-requirement.yaml" <<EOF
apiVersion: apiextensions.openshift.io/v1alpha1
kind: CompatibilityRequirement
metadata:
  name: tc03-real-test-protection
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

oc apply -f "$BACKUP_DIR/compatibility-requirement.yaml"

echo "✓ CompatibilityRequirement created"
```

### Step 2: Wait and Verify Status

```bash
echo "=== Waiting for Reconciliation ==="
echo "Waiting 15 seconds for controller to process..."
sleep 15

ADMITTED=$(oc get compatibilityrequirement tc03-real-test-protection \
  -o jsonpath='{.status.conditions[?(@.type=="Admitted")].status}')
COMPATIBLE=$(oc get compatibilityrequirement tc03-real-test-protection \
  -o jsonpath='{.status.conditions[?(@.type=="Compatible")].status}')

echo ""
echo "Status Check:"
echo "  Admitted: $ADMITTED"
echo "  Compatible: $COMPATIBLE"

if [ "$ADMITTED" != "True" ] || [ "$COMPATIBLE" != "True" ]; then
    echo ""
    echo "❌ CompatibilityRequirement not ready!"
    echo "Status is not Admitted=True and Compatible=True"
    echo ""
    echo "Full status:"
    oc get compatibilityrequirement tc03-real-test-protection -o yaml
    echo ""
    echo "Cannot proceed with test. Cleaning up..."
    oc delete compatibilityrequirement tc03-real-test-protection
    exit 1
fi

echo "✓ CompatibilityRequirement is ready"
```

### Step 3: Create Modified CRD (Remove replicas field)

```bash
echo "=== Creating Modified CRD Version ==="

yq eval 'del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas)' \
  "$BACKUP_DIR/machineset-crd-backup.yaml" > "$BACKUP_DIR/machineset-crd-modified.yaml"

# Verify field was removed
ORIGINAL_HAS_REPLICAS=$(grep -c "replicas:" "$BACKUP_DIR/machineset-crd-backup.yaml" || echo "0")
MODIFIED_HAS_REPLICAS=$(grep -c "replicas:" "$BACKUP_DIR/machineset-crd-modified.yaml" || echo "0")

echo "Original CRD has replicas field: $ORIGINAL_HAS_REPLICAS occurrences"
echo "Modified CRD has replicas field: $MODIFIED_HAS_REPLICAS occurrences"

if [ $MODIFIED_HAS_REPLICAS -ge $ORIGINAL_HAS_REPLICAS ]; then
    echo "❌ Field was not removed properly!"
    exit 1
fi

echo "✓ Modified CRD created with replicas field removed"
```

### Step 4: 🔥 CRITICAL TEST - Attempt Real Modification

```bash
echo ""
echo "========================================="
echo "    CRITICAL TEST: Real CRD Modification"
echo "========================================="
echo ""
echo "⚠️ This will ACTUALLY attempt to modify the CRD"
echo "⚠️ Webhook MUST reject this or CRD will be broken"
echo ""
read -p "Press Enter to proceed or Ctrl+C to abort..."
echo ""

# Attempt the modification (NO DRY-RUN!)
echo "Executing: oc apply -f machineset-crd-modified.yaml"
echo ""

if oc apply -f "$BACKUP_DIR/machineset-crd-modified.yaml" 2>&1 | tee "$BACKUP_DIR/apply-result.txt"; then
    echo ""
    echo "═══════════════════════════════════════"
    echo "❌❌❌ TEST FAILED! ❌❌❌"
    echo "═══════════════════════════════════════"
    echo ""
    echo "CRITICAL: Modification was ALLOWED!"
    echo "Webhook did NOT reject the field removal!"
    echo ""
    echo "The CRD may now be broken!"
    echo ""

    # Immediate recovery attempt
    echo "=== ATTEMPTING IMMEDIATE RECOVERY ==="
    echo ""

    if oc apply -f "$BACKUP_DIR/machineset-crd-backup.yaml"; then
        echo "✓ CRD restored from backup"
    else
        echo "❌ FAILED to restore CRD!"
        echo "Manual intervention required!"
        echo "Backup location: $BACKUP_DIR/machineset-crd-backup.yaml"
    fi

    # Cleanup
    oc delete compatibilityrequirement tc03-real-test-protection

    exit 1
else
    # Modification was REJECTED - this is SUCCESS!
    EXIT_CODE=$?

    echo ""
    echo "═══════════════════════════════════════"
    echo "✅✅✅ TEST PASSED! ✅✅✅"
    echo "═══════════════════════════════════════"
    echo ""
    echo "Exit code: $EXIT_CODE (non-zero = rejected)"
    echo ""
    echo "Webhook correctly REJECTED the field removal!"
    echo ""
    echo "Rejection message:"
    echo "---"
    cat "$BACKUP_DIR/apply-result.txt"
    echo "---"
    echo ""
fi
```

## ✅ Verification Steps

### Verify 1: Check CRD Was NOT Modified

```bash
echo "=== Verifying CRD Was Not Modified ==="

CURRENT_VERSION=$(oc get crd machinesets.cluster.x-k8s.io \
  -o jsonpath='{.metadata.resourceVersion}')
CURRENT_GENERATION=$(oc get crd machinesets.cluster.x-k8s.io \
  -o jsonpath='{.metadata.generation}')

echo "Original ResourceVersion: $ORIGINAL_VERSION"
echo "Current ResourceVersion:  $CURRENT_VERSION"
echo ""
echo "Original Generation: $ORIGINAL_GENERATION"
echo "Current Generation:  $CURRENT_GENERATION"

if [ "$CURRENT_VERSION" == "$ORIGINAL_VERSION" ] && [ "$CURRENT_GENERATION" == "$ORIGINAL_GENERATION" ]; then
    echo ""
    echo "✓ CRD was NOT modified (ResourceVersion and Generation unchanged)"
else
    echo ""
    echo "⚠️ CRD metadata changed!"
    echo "This might indicate the CRD was modified or reconciled"
    echo "Checking field content..."
fi
```

### Verify 2: Check replicas Field Still Exists

```bash
echo ""
echo "=== Verifying replicas Field Still Exists ==="

CURRENT_HAS_REPLICAS=$(oc get crd machinesets.cluster.x-k8s.io -o yaml | grep -c "replicas:" || echo "0")

echo "Current CRD has replicas field: $CURRENT_HAS_REPLICAS occurrences"
echo "Expected (original):            $ORIGINAL_HAS_REPLICAS occurrences"

if [ $CURRENT_HAS_REPLICAS -ge $ORIGINAL_HAS_REPLICAS ]; then
    echo "✓ replicas field still exists - CRD protected successfully"
else
    echo "❌ replicas field is missing - CRD was modified!"
    echo "Attempting recovery..."
    oc apply -f "$BACKUP_DIR/machineset-crd-backup.yaml"
fi
```

### Verify 3: Check Existing Resources Still Work

```bash
echo ""
echo "=== Verifying Existing MachineSet Resources ==="

if [ $MACHINESET_COUNT -gt 0 ]; then
    # Try to get a MachineSet to verify CRD is still functional
    if oc get machinesets -A --no-headers | head -1 &> /dev/null; then
        echo "✓ MachineSet resources are still accessible"
        echo "✓ CRD is functional"
    else
        echo "❌ Cannot access MachineSet resources!"
        echo "CRD may be broken - attempting recovery..."
        oc apply -f "$BACKUP_DIR/machineset-crd-backup.yaml"
    fi
else
    echo "ℹ No existing MachineSet resources to verify"
fi
```

### Verify 4: Check Webhook Logs

```bash
echo ""
echo "=== Checking Webhook Logs ==="

OPERATOR_POD=$(oc get pods -n openshift-compatibility-requirements-operator \
  -o jsonpath='{.items[0].metadata.name}')

echo "Webhook pod: $OPERATOR_POD"
echo ""
echo "Recent webhook activity:"
oc logs -n openshift-compatibility-requirements-operator $OPERATOR_POD --tail=20 | \
  grep -E "webhook|deny|reject|machinesets" || echo "No relevant logs found"
```

## 🧹 Cleanup

```bash
echo ""
echo "=== Cleaning Up Test Resources ==="

# Delete CompatibilityRequirement
oc delete compatibilityrequirement tc03-real-test-protection

echo "✓ CompatibilityRequirement deleted"

# Keep backup for safety
echo ""
echo "Backup location: $BACKUP_DIR"
echo "You can delete this manually when safe:"
echo "  rm -rf $BACKUP_DIR"
```

## 🆘 Emergency Recovery

If the CRD was modified and broken:

### Recovery Option 1: Restore from Backup

```bash
echo "=== EMERGENCY: Restoring CRD from Backup ==="

BACKUP_FILE=$(cat /tmp/last-crd-backup.txt)

if [ -f "$BACKUP_FILE" ]; then
    echo "Restoring from: $BACKUP_FILE"
    oc apply -f "$BACKUP_FILE"

    echo "Waiting for CRD to re-establish..."
    sleep 10

    # Verify restoration
    if oc get crd machinesets.cluster.x-k8s.io &> /dev/null; then
        echo "✓ CRD restored successfully"
    else
        echo "❌ CRD restoration failed - manual intervention needed"
    fi
else
    echo "❌ Backup file not found!"
    echo "You need to restore from etcd backup or reinstall"
fi
```

### Recovery Option 2: Check if MachineSet Operator Can Restore

```bash
# The MachineSet operator might reconcile and restore the CRD
echo "Checking if operator can self-heal..."

# Wait for operator to notice and reconcile
sleep 30

oc get crd machinesets.cluster.x-k8s.io -o yaml | grep -A 5 "replicas:"
```

### Recovery Option 3: Reinstall from Source

```bash
# If you know where the CRD comes from
# For CAPI, it might be cluster-api-operator

# Check what manages this CRD
oc get crd machinesets.cluster.x-k8s.io -o yaml | grep -E "ownerReferences|annotations" | head -20
```

## 📊 Test Results Checklist

After completing the test, verify:

- [ ] ✅ Modification was REJECTED by Webhook
- [ ] ✅ CRD ResourceVersion unchanged
- [ ] ✅ CRD Generation unchanged
- [ ] ✅ replicas field still exists in CRD
- [ ] ✅ Existing MachineSet resources still accessible
- [ ] ✅ Webhook logs show rejection
- [ ] ✅ Error message mentions "compatibility"
- [ ] ✅ Backup created and verified
- [ ] ✅ CompatibilityRequirement cleaned up

## 🎓 What Success Looks Like

**Expected output when test PASSES:**

```
========================================
    CRITICAL TEST: Real CRD Modification
========================================

⚠️ This will ACTUALLY attempt to modify the CRD
⚠️ Webhook MUST reject this or CRD will be broken

Executing: oc apply -f machineset-crd-modified.yaml

Error from server: admission webhook "compatibilityrequirement.apiextensions.openshift.io" denied the request:
removing field 'spec.replicas' from version 'v1beta1' would break compatibility requirement tc03-real-test-protection

═══════════════════════════════════════
✅✅✅ TEST PASSED! ✅✅✅
═══════════════════════════════════════

Exit code: 1 (non-zero = rejected)

Webhook correctly REJECTED the field removal!
```

**Key indicators of success:**
1. `Error from server` - Kubernetes rejected the request
2. `admission webhook ... denied the request` - Webhook was the rejector
3. `would break compatibility requirement` - Correct reason
4. Exit code is non-zero (failure to apply is success for this test!)

## 📝 Complete Test Script

Save this as `tc03-real-test-no-dryrun.sh`:

```bash
#!/bin/bash
# Real TC-03 test without dry-run
# ⚠️ WARNING: This ACTUALLY modifies the CRD if Webhook fails!
# ONLY use in test clusters!

set -e

# [Include all the code sections above in sequence]
```

## 🔒 Safety Checklist Before Running

Before you run the real test:

- [ ] ✅ Confirmed this is a TEST cluster (not production)
- [ ] ✅ Have cluster admin access
- [ ] ✅ Created full CRD backup
- [ ] ✅ Verified backup file is valid
- [ ] ✅ Know how to restore from backup
- [ ] ✅ Verified CompatibilityRequirement Status is Admitted=True
- [ ] ✅ Webhook pod is running
- [ ] ✅ Have time to recover if something goes wrong
- [ ] ✅ No critical workloads depend on this cluster RIGHT NOW

---

**Remember**: With great power comes great responsibility. The `--dry-run=server` mode exists for a reason! 🛡️
