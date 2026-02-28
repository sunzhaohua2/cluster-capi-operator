# Analysis: Apply Succeeded But Field Still Exists

## 🔍 Your Test Result

```bash
$ oc apply -f machineset-crd-without-replicas.yaml

Warning: resource customresourcedefinitions/machinesets.cluster.x-k8s.io is missing the
kubectl.kubernetes.io/last-applied-configuration annotation which is required by oc apply.
customresourcedefinition.apiextensions.k8s.io/machinesets.cluster.x-k8s.io configured

# But checking the CRD:
$ oc get crd machinesets.cluster.x-k8s.io -o yaml | grep -A 7 replicas:
replicas:
  description: |-
    replicas is the number of desired replicas.
    This is a pointer to distinguish between explicit zero and unspecified.
    Defaults to 1.
  format: int32
  type: integer
```

**Summary:**
- ✅ `oc apply` command succeeded (exit code 0)
- ✅ Output shows `configured`
- ✅ But `replicas` field is STILL THERE!

## 🤔 What Could This Mean?

### Hypothesis 1: Webhook Blocked But Apply Reported Success

**Possibility:** The Webhook rejected the change, but `oc apply` still reported success because it compared what was requested vs. what exists and saw no difference.

**Check this:**
```bash
# Check ResourceVersion - did it actually change?
oc get crd machinesets.cluster.x-k8s.io -o jsonpath='{.metadata.resourceVersion}'

# Compare with before the apply
# If ResourceVersion is the SAME, nothing was actually modified
```

### Hypothesis 2: Server-Side Apply Behavior

**Possibility:** Server-side apply might handle field removal differently, especially with the warning about missing `last-applied-configuration` annotation.

**The warning suggests:**
- This CRD was not created with `oc apply`
- It might have been created by an operator or helm
- `oc apply` behavior might be different in this case

### Hypothesis 3: CRD Controller Auto-Reconciled

**Possibility:** The modification went through, but the CAPI operator immediately reconciled it back.

**Timeline would be:**
1. Your apply removes the field
2. CRD gets modified
3. CAPI operator detects change
4. CAPI operator restores the correct schema
5. By the time you check, it's already restored

**Check this:**
```bash
# Check CRD events
oc get events --field-selector involvedObject.name=machinesets.cluster.x-k8s.io \
  --sort-by='.lastTimestamp' | tail -20

# Look for recent updates to the CRD
```

### Hypothesis 4: Webhook Silently Allowed (Warn Mode?)

**Possibility:** The Webhook is in Warn mode instead of Deny mode.

**Check this:**
```bash
# Check your CompatibilityRequirement action
oc get compatibilityrequirement -o yaml | grep -A 2 "action:"

# Should see:
#   action: Deny  (rejects)
# Not:
#   action: Warn  (allows with warning)
```

## 🔬 Detailed Investigation Steps

### Step 1: Check if CRD Actually Changed

```bash
echo "=== Checking CRD Modification ==="

# Get current CRD and save
oc get crd machinesets.cluster.x-k8s.io -o yaml > /tmp/crd-current.yaml

# Check resourceVersion (this increments with each modification)
RESOURCE_VERSION=$(oc get crd machinesets.cluster.x-k8s.io \
  -o jsonpath='{.metadata.resourceVersion}')

echo "Current ResourceVersion: $RESOURCE_VERSION"

# Check generation (this increments with spec changes)
GENERATION=$(oc get crd machinesets.cluster.x-k8s.io \
  -o jsonpath='{.metadata.generation}')

echo "Current Generation: $GENERATION"

# If you noted these values BEFORE your apply, compare them
# If they're the SAME, the CRD was never actually modified
```

### Step 2: Try Again with Explicit ResourceVersion

```bash
echo "=== Testing Again with Verbose Output ==="

# Try the apply again with verbose output
oc apply -f machineset-crd-without-replicas.yaml -v=8 2>&1 | tee /tmp/apply-verbose.log

# Look for:
# - "Patch" operations
# - Webhook calls
# - Validation messages
```

### Step 3: Check Webhook Logs During Apply

```bash
echo "=== Checking Webhook Logs ==="

# Get operator pod
OPERATOR_POD=$(oc get pods -n openshift-compatibility-requirements-operator \
  -o jsonpath='{.items[0].metadata.name}')

# Follow logs while you apply
echo "Watching logs... (in another terminal, run the apply)"
oc logs -f -n openshift-compatibility-requirements-operator $OPERATOR_POD

# OR check recent logs
oc logs -n openshift-compatibility-requirements-operator $OPERATOR_POD --tail=50 | \
  grep -A 5 -B 5 "machinesets"
```

### Step 4: Check CompatibilityRequirement Status

```bash
echo "=== Checking CompatibilityRequirement ==="

# Make sure it still exists and is active
oc get compatibilityrequirement

# Check its status
oc get compatibilityrequirement <your-cr-name> -o yaml | \
  grep -A 20 "status:"

# Specifically check action mode
oc get compatibilityrequirement <your-cr-name> \
  -o jsonpath='{.spec.compatibilitySchema.customResourceDefinitionSchemaValidation.action}'

echo ""
echo "Action should be: Deny"
```

### Step 5: Check Webhook Configuration

```bash
echo "=== Checking Webhook Configuration ==="

# Check if webhook exists
oc get validatingwebhookconfiguration | grep compatibility

# Check webhook details
WEBHOOK_NAME="openshift-compatibility-requirements-apiextensions-k8s-io-v1-customresourcedefinition-validation"

oc get validatingwebhookconfiguration $WEBHOOK_NAME -o yaml | \
  grep -A 5 "failurePolicy\|rules\|operations"

# Should show:
# - operations: ["UPDATE", "CREATE"]
# - failurePolicy: Fail (or Ignore)
```

### Step 6: Understand the Warning Message

The warning you saw:
```
Warning: resource customresourcedefinitions/machinesets.cluster.x-k8s.io is missing the
kubectl.kubernetes.io/last-applied-configuration annotation
```

**This means:**
- The CRD was NOT created with `oc apply` or `kubectl apply`
- It was likely created by an operator (cluster-api-operator)
- `oc apply` will use **three-way merge** differently

**Impact:**
When `last-applied-configuration` is missing, `oc apply`:
1. Might behave more like `oc replace`
2. Might not detect field removals correctly
3. Might report success even if server rejected changes

## 🎯 Most Likely Explanation

Based on your result, here's what probably happened:

### Scenario A: Webhook Did Its Job (Silently)

```
1. You run: oc apply -f machineset-crd-without-replicas.yaml
2. Request goes to API server
3. API server calls Webhook
4. Webhook sees: replicas field being removed
5. Webhook returns: DENY (reject)
6. API server rejects the modification
7. BUT oc apply doesn't show an error because:
   - It compared desired state vs actual state
   - They're "equivalent" in some way
   - Reports "configured" even though nothing changed
8. Field still exists because modification was rejected
```

**To confirm:**
Check if ResourceVersion changed. If it didn't change = nothing was modified.

### Scenario B: Server-Side Apply Protected It

```
1. oc apply sends the modification
2. Server-side apply logic sees you're removing a field
3. But the field is managed by another field manager (the operator)
4. Server-side apply refuses to remove it (field ownership conflict)
5. Reports success but doesn't actually remove the field
```

**To confirm:**
```bash
oc get crd machinesets.cluster.x-k8s.io -o yaml | grep -A 5 "managedFields"
# Look for who manages the "replicas" field
```

## ✅ Verification Test

Let's definitively test if Webhook is working:

```bash
#!/bin/bash
# Definitive Webhook test

echo "=== Before Modification ==="
BEFORE_VERSION=$(oc get crd machinesets.cluster.x-k8s.io \
  -o jsonpath='{.metadata.resourceVersion}')
BEFORE_GEN=$(oc get crd machinesets.cluster.x-k8s.io \
  -o jsonpath='{.metadata.generation}')

echo "ResourceVersion: $BEFORE_VERSION"
echo "Generation: $BEFORE_GEN"

echo ""
echo "=== Attempting Modification ==="
oc apply -f machineset-crd-without-replicas.yaml

echo ""
echo "=== After Modification ==="
AFTER_VERSION=$(oc get crd machinesets.cluster.x-k8s.io \
  -o jsonpath='{.metadata.resourceVersion}')
AFTER_GEN=$(oc get crd machinesets.cluster.x-k8s.io \
  -o jsonpath='{.metadata.generation}')

echo "ResourceVersion: $AFTER_VERSION"
echo "Generation: $AFTER_GEN"

echo ""
echo "=== Analysis ==="
if [ "$BEFORE_VERSION" = "$AFTER_VERSION" ] && [ "$BEFORE_GEN" = "$AFTER_GEN" ]; then
    echo "✅ CRD WAS NOT MODIFIED"
    echo "ResourceVersion and Generation are UNCHANGED"
    echo ""
    echo "This means:"
    echo "  - Either Webhook blocked it"
    echo "  - Or server-side apply prevented it"
    echo "  - Or apply detected no actual change needed"
    echo ""
    echo "The 'configured' message is misleading!"
else
    echo "❌ CRD WAS MODIFIED!"
    echo "ResourceVersion changed: $BEFORE_VERSION → $AFTER_VERSION"
    echo "Generation changed: $BEFORE_GEN → $AFTER_GEN"
    echo ""
    echo "This means Webhook did NOT block the change!"
fi

echo ""
echo "=== Checking replicas field ==="
if oc get crd machinesets.cluster.x-k8s.io -o yaml | grep -q "replicas:"; then
    echo "✅ replicas field still exists"
else
    echo "❌ replicas field is MISSING!"
fi
```

## 🎓 Understanding `oc apply` Behavior

`oc apply` can show `configured` even when nothing changed because:

1. **Declarative nature**: It declares desired state, not commands
2. **Comparison logic**: Compares desired vs actual
3. **Success criteria**: "State matches desired" = success
4. **Even if rejected**: If end state matches, reports success

This is different from `oc replace` or `oc patch` which would show errors if rejected.

## 📝 Recommended Next Steps

1. **Run the verification test above** - Check ResourceVersion
2. **Check Webhook logs** - See if there's any rejection logged
3. **Test with a different field** - Try removing a less critical field
4. **Use `oc replace` instead** - It will show errors more clearly:
   ```bash
   oc replace -f machineset-crd-without-replicas.yaml
   ```

## 💡 Bottom Line

**Your Webhook might actually be working!**

The fact that:
- Command says "configured"
- But field still exists

Could mean:
- ✅ Webhook silently blocked it
- ✅ Server-side apply protected it
- ✅ CRD was never actually modified

**To know for sure**: Check if ResourceVersion changed!

---

**Quick check:**
```bash
# If this shows the same value before and after your apply:
oc get crd machinesets.cluster.x-k8s.io -o jsonpath='{.metadata.resourceVersion}'

# Then nothing was actually modified, and Webhook (or server-side apply) DID protect it!
```
