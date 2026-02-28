# TC-04 快速开始指南

## 一句话总结

**TC-04 测试 `excludedFields` 功能（PR #467）：允许删除 CompatibilityRequirement 中明确排除的字段，而其他字段仍受保护。**

## 5 分钟快速测试

### 前提条件

```bash
# 1. 设置 KUBECONFIG
export KUBECONFIG=/path/to/your/kubeconfig

# 2. 确认集群已部署 cluster-capi-operator (PR #467 分支)
oc get deployment -n openshift-cluster-api crd-compatibility-checker

# 3. 确认 MachineSet CRD 存在
oc get crd machinesets.cluster.x-k8s.io
```

### 快速运行（自动化）

```bash
cd /Users/zhsun/go/src/github.com/openshift/cluster-capi-operator

# 一键运行完整测试
./test-manifests/tc04-excluded-fields-test.sh
```

**预期输出：**
```
✅ All tests PASSED! excludedFields feature is working correctly.
```

### 核心测试步骤（手动）

如果自动化脚本失败，可以手动测试核心功能：

#### 步骤 1: 创建带 excludedFields 的 CompatibilityRequirement

```bash
# 获取当前 CRD
oc get crd machinesets.cluster.x-k8s.io -o yaml | sed 's/^/        /' > /tmp/crd-indented.yaml

# 获取 storage version
STORAGE_VERSION=$(oc get crd machinesets.cluster.x-k8s.io -o jsonpath='{.spec.versions[?(@.storage==true)].name}')

# 创建 CR（排除 spec.replicas 字段）
cat <<EOF | oc apply -f -
apiVersion: apiextensions.openshift.io/v1alpha1
kind: CompatibilityRequirement
metadata:
  name: tc04-test
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

#### 步骤 2: 删除排除的字段（应该成功）

```bash
# 备份原始 CRD
oc get crd machinesets.cluster.x-k8s.io -o yaml > /tmp/original-crd.yaml

# 删除 spec.replicas 字段（需要 yq 工具）
yq eval 'del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas)' \
  /tmp/original-crd.yaml | oc apply -f -
```

**✅ 预期结果：**
```
customresourcedefinition.apiextensions.k8s.io/machinesets.cluster.x-k8s.io configured
```

#### 步骤 3: 验证字段已删除

```bash
# 应该返回空（字段不存在）
oc get crd machinesets.cluster.x-k8s.io \
  -o jsonpath='{.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas}'
```

#### 步骤 4: 清理

```bash
# 恢复原始 CRD
oc apply -f /tmp/original-crd.yaml

# 删除测试 CR
oc delete compatibilityrequirement tc04-test
```

## 与 TC-03 的关键区别

| 测试 | excludedFields | 删除 spec.replicas 结果 |
|------|----------------|------------------------|
| **TC-03** | ❌ 不使用 | ❌ **拒绝** |
| **TC-04** | ✅ 排除 spec.replicas | ✅ **允许** |

## 验证点 Checklist

- [ ] CompatibilityRequirement 创建成功
- [ ] Status 显示 Admitted=True
- [ ] excludedFields 配置正确
- [ ] 删除排除的字段成功（spec.replicas）
- [ ] 字段确实从 CRD 中消失
- [ ] ResourceVersion 发生变化
- [ ] 删除非排除的字段仍然被拒绝（spec.selector）

## 常见错误

### 1. "field spec.replicas is required but missing"

**原因：** excludedFields 配置错误或未生效

**检查：**
```bash
# 验证 excludedFields 配置
oc get compatibilityrequirement tc04-test \
  -o jsonpath='{.spec.compatibilitySchema.excludedFields}' | jq .

# 应该看到：
# [{"path": "spec.replicas", "versions": ["v1beta1"]}]
```

### 2. "admission webhook denied the request"

**原因：** 尝试删除的字段不在 excludedFields 中

**这是正常的！** 说明 webhook 正在保护非排除的字段。

### 3. CR Status Admitted=False

**原因：** CompatibilityRequirement 配置有语法错误

**检查：**
```bash
oc get compatibilityrequirement tc04-test -o yaml | grep -A 20 status
```

## 高级用法

### 排除多个字段

```yaml
excludedFields:
  - path: "spec.replicas"
  - path: "spec.minReadySeconds"
  - path: "status.conditions.lastTransitionTime"
```

### 版本特定排除

```yaml
excludedFields:
  - path: "spec.replicas"
    versions:
      - v1beta1  # 只在 v1beta1 中排除
  - path: "status.deprecated"
    # 不指定 versions - 在所有版本中排除
```

### 排除嵌套字段

```yaml
excludedFields:
  - path: "status.conditions.lastTransitionTime"  # 嵌套字段
  - path: "spec.template.spec.volumes.name"      # 深层嵌套
```

## 实际应用场景

### 场景：CAPI 版本升级时废弃字段

```yaml
# HyperShift 想升级 CAPI 到新版本，删除 deprecated 字段
# 但 OpenShift 控制器仍需要其他字段

apiVersion: apiextensions.openshift.io/v1alpha1
kind: CompatibilityRequirement
metadata:
  name: capi-machine-openshift
spec:
  compatibilitySchema:
    customResourceDefinition:
      name: machines.cluster.x-k8s.io
      type: YAML
      data: |
        # OpenShift 需要的 v1beta1 版本
    requiredVersions:
      defaultSelection: StorageOnly
    excludedFields:
      - path: "status.deprecated"  # 允许 HyperShift 删除这个字段
      - path: "status.v1beta1Deprecated"
  customResourceDefinitionSchemaValidation:
    action: Deny
```

**结果：**
- ✅ HyperShift 可以升级到删除了 deprecated 字段的新 CRD
- ✅ OpenShift 需要的其他字段仍然受保护
- ✅ 升级过程平滑，没有 breaking change

## 更多信息

- **完整手册：** [MANUAL_TEST_TC04_EXCLUDED_FIELDS.md](MANUAL_TEST_TC04_EXCLUDED_FIELDS.md)
- **对比文档：** [TC03_VS_TC04_COMPARISON.md](TC03_VS_TC04_COMPARISON.md)
- **测试索引：** [TEST_INDEX.md](TEST_INDEX.md)
- **PR #467：** https://github.com/openshift/cluster-capi-operator/pull/467

## 获取帮助

遇到问题？运行诊断脚本：

```bash
cd test-manifests
./diagnose-finalizer.sh
```

检查 webhook 日志：

```bash
oc logs -n openshift-cluster-api \
  -l app=crd-compatibility-checker \
  --tail=100 | grep -i "excluded\|replicas"
```
