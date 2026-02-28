# TC-03 vs TC-04: 测试对比

## 概述

这两个测试用例都测试 CRD 字段删除场景，但行为完全相反：

| 测试用例 | 功能 | 删除字段结果 | 测试的 PR |
|---------|------|-------------|-----------|
| **TC-03** | 字段删除保护 | ❌ **拒绝** | PR #459 |
| **TC-04** | 排除字段（允许删除） | ✅ **允许** | PR #467 |

## 核心区别

### TC-03: Field Removal Protection (字段删除保护)

```yaml
# CompatibilityRequirement 配置
spec:
  compatibilitySchema:
    customResourceDefinition:
      data: |
        # 包含 spec.replicas 字段的完整 CRD
    requiredVersions:
      defaultSelection: StorageOnly
    # ❌ 没有 excludedFields
  customResourceDefinitionSchemaValidation:
    action: Deny
```

**测试场景：**
```bash
# 尝试删除 spec.replicas 字段
oc apply -f machineset-crd-without-replicas.yaml
```

**预期结果：**
```
❌ Error from server: admission webhook "vcrd.openshift.io" denied the request:
   field spec.replicas is required but missing in the updated CRD
```

**用途：** 保护关键字段不被意外删除

---

### TC-04: Excluded Fields (排除字段功能)

```yaml
# CompatibilityRequirement 配置
spec:
  compatibilitySchema:
    customResourceDefinition:
      data: |
        # 包含 spec.replicas 字段的完整 CRD
    requiredVersions:
      defaultSelection: StorageOnly
    # ✅ 添加了 excludedFields
    excludedFields:
      - path: "spec.replicas"
        versions:
          - v1beta1
  customResourceDefinitionSchemaValidation:
    action: Deny
```

**测试场景：**
```bash
# 尝试删除 spec.replicas 字段（已在 excludedFields 中）
oc apply -f machineset-crd-without-replicas.yaml
```

**预期结果：**
```
✅ customresourcedefinition.apiextensions.k8s.io/machinesets.cluster.x-k8s.io configured
```

**用途：** 允许废弃字段的优雅删除

## 详细对比表

| 方面 | TC-03 | TC-04 |
|------|-------|-------|
| **测试的 PR** | #459 (基础功能) | #467 (excludedFields) |
| **配置** | 无 excludedFields | 有 excludedFields |
| **删除 spec.replicas** | ❌ 被拒绝 | ✅ 被允许 |
| **删除 spec.selector** | ❌ 被拒绝 | ❌ 被拒绝（未排除） |
| **Webhook 行为** | 严格验证所有字段 | 忽略排除的字段 |
| **使用场景** | 强制兼容性保护 | 字段废弃/迁移 |
| **ResourceVersion** | 不变（被拒绝） | 改变（被允许修改） |

## 测试流程对比

### TC-03 测试流程

```bash
# 1. 创建 CompatibilityRequirement（无 excludedFields）
oc apply -f tc03-compatibility-requirement.yaml

# 2. 尝试删除字段
oc apply -f machineset-crd-without-replicas.yaml
# → ❌ 失败（预期行为）

# 3. 验证 ResourceVersion 未改变
# → 确认 CRD 未被修改
```

### TC-04 测试流程

```bash
# 1. 创建 CompatibilityRequirement（有 excludedFields）
oc apply -f tc04-compatibility-requirement.yaml

# 2. 尝试删除排除的字段
oc apply -f machineset-crd-without-replicas.yaml
# → ✅ 成功（预期行为）

# 3. 验证 ResourceVersion 已改变
# → 确认 CRD 已被修改

# 4. 尝试删除非排除的字段
oc apply -f machineset-crd-without-selector.yaml
# → ❌ 失败（预期行为，selector 未排除）
```

## 实际使用场景

### 场景 1: 严格保护（使用 TC-03 方式）

**情况：** OpenShift 核心组件依赖 CAPI Machine CRD 的 v1beta1 版本

```yaml
# cluster-capi-operator 创建的 CompatibilityRequirement
apiVersion: apiextensions.openshift.io/v1alpha1
kind: CompatibilityRequirement
metadata:
  name: machine-ccapio-requirement
spec:
  compatibilitySchema:
    customResourceDefinition:
      data: |
        # Machine CRD v1beta1 完整定义
    requiredVersions:
      defaultSelection: StorageOnly
    # 不使用 excludedFields - 保护所有字段
  customResourceDefinitionSchemaValidation:
    action: Deny
```

**效果：**
- HyperShift 尝试升级到只有 v1beta2 的 CRD → ❌ 被拒绝
- HyperShift 升级到同时包含 v1beta1 和 v1beta2 的 CRD → ✅ 允许

---

### 场景 2: 优雅废弃字段（使用 TC-04 方式）

**情况：** CAPI v1beta2 引入了 `status.deprecated` 字段，将在未来版本删除

```yaml
# cluster-capi-operator 创建的 CompatibilityRequirement
apiVersion: apiextensions.openshift.io/v1alpha1
kind: CompatibilityRequirement
metadata:
  name: machine-ccapio-requirement
spec:
  compatibilitySchema:
    customResourceDefinition:
      data: |
        # Machine CRD 包含 status.deprecated 字段
    requiredVersions:
      defaultSelection: StorageOnly
    # 使用 excludedFields - 允许删除 deprecated 字段
    excludedFields:
      - path: "status.deprecated"
        versions:
          - v1beta2
  customResourceDefinitionSchemaValidation:
    action: Deny
```

**效果：**
- HyperShift 升级到删除了 `status.deprecated` 的新 CRD → ✅ 允许
- HyperShift 尝试删除 `spec.replicas` → ❌ 被拒绝（未排除）

## 如何选择测试用例

### 选择 TC-03（字段保护）当：

- ✅ 测试基本的 CRD 兼容性保护功能
- ✅ 验证 Webhook 能正确拒绝不兼容的 CRD 更新
- ✅ 确保关键字段不会被意外删除
- ✅ 测试 PR #459 的核心功能

### 选择 TC-04（排除字段）当：

- ✅ 测试字段废弃/迁移场景
- ✅ 验证 `excludedFields` API 功能
- ✅ 测试 PR #467 的新增功能
- ✅ 模拟 CAPI 升级时废弃字段的处理

### 两个都测试当：

- ✅ 完整验证 CRD Compatibility Checker 功能
- ✅ 回归测试：确保 TC-04 不影响 TC-03 的保护功能
- ✅ 准备生产部署前的完整测试

## 快速参考

### TC-03 快速运行

```bash
cd /Users/zhsun/go/src/github.com/openshift/cluster-capi-operator

# 手动测试
cat MANUAL_TEST_TC03_REMOVE_FIELD.md

# 自动化测试
./test-manifests/tc03-manual-test.sh
```

### TC-04 快速运行

```bash
cd /Users/zhsun/go/src/github.com/openshift/cluster-capi-operator

# 手动测试
cat MANUAL_TEST_TC04_EXCLUDED_FIELDS.md

# 自动化测试
./test-manifests/tc04-excluded-fields-test.sh
```

### 两个都运行（完整测试）

```bash
cd /Users/zhsun/go/src/github.com/openshift/cluster-capi-operator

# 先运行 TC-03
./test-manifests/tc03-manual-test.sh

# 等待清理完成，然后运行 TC-04
./test-manifests/tc04-excluded-fields-test.sh
```

## API 字段对比

### CompatibilityRequirement Spec

```yaml
spec:
  compatibilitySchema:
    customResourceDefinition:
      # TC-03 ✅ | TC-04 ✅
      name: machinesets.cluster.x-k8s.io
      type: YAML
      data: |
        <CRD YAML>

    requiredVersions:
      # TC-03 ✅ | TC-04 ✅
      defaultSelection: StorageOnly

    excludedFields:
      # TC-03 ❌ 不使用
      # TC-04 ✅ 使用
      - path: "spec.replicas"
        versions:
          - v1beta1

  customResourceDefinitionSchemaValidation:
    # TC-03 ✅ | TC-04 ✅
    action: Deny
```

## 测试矩阵

| 操作 | TC-03<br/>（无 excludedFields） | TC-04<br/>（有 excludedFields） |
|------|------------------------------|-------------------------------|
| 创建 CompatibilityRequirement | ✅ 成功 | ✅ 成功 |
| CR Status Admitted=True | ✅ 是 | ✅ 是 |
| CR Status Compatible=True | ✅ 是 | ✅ 是 |
| 删除 spec.replicas | ❌ **拒绝** | ✅ **允许** |
| 删除 spec.selector | ❌ **拒绝** | ❌ **拒绝** |
| ResourceVersion 改变 | ❌ 否 | ✅ 是（删除 replicas 时） |
| Webhook 日志 | "field required" | "field excluded from validation" |

## 常见问题

### Q: TC-04 会影响 TC-03 的保护功能吗？

A: 不会。`excludedFields` 只影响明确列出的字段。未列出的字段仍然受到 TC-03 同样的保护。

### Q: 可以先运行 TC-03，然后直接运行 TC-04 吗？

A: 可以，但建议让 TC-03 清理完成后再运行 TC-04。或者手动删除 TC-03 创建的 CompatibilityRequirement：
```bash
oc delete compatibilityrequirement tc03-machineset-protection
```

### Q: excludedFields 支持通配符吗？

A: 不支持。必须明确指定完整的字段路径（如 `spec.replicas`，不能用 `spec.*`）。

### Q: 如何排除嵌套字段？

A: 使用点分隔路径，例如：
```yaml
excludedFields:
  - path: "status.conditions.lastTransitionTime"
  - path: "spec.template.spec.volumes.name"
```

### Q: 能排除数组中的特定元素吗？

A: 不能。对于数组字段（如 `status.conditions`），排除会应用到数组中的所有元素。

## 总结

| 方面 | TC-03 | TC-04 |
|------|-------|-------|
| **目标** | 保护字段不被删除 | 允许特定字段被删除 |
| **复杂度** | 简单 | 中等（需配置 excludedFields） |
| **用途** | 基础兼容性保护 | 字段废弃/迁移 |
| **生产应用** | 所有 CRD 保护场景 | CAPI 版本升级场景 |
| **测试优先级** | ⭐⭐⭐⭐⭐ 必须 | ⭐⭐⭐⭐ 重要 |

两个测试用例互补，共同验证 CRD Compatibility Checker 的完整功能。
