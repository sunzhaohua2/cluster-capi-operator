# PR #459 测试计划

## 📋 概述

本测试计划用于验证 PR https://github.com/openshift/cluster-capi-operator/pull/459 实现的功能。

### 测试范围

根据PR描述和EP要求，当前PR实现了以下功能：

✅ **已实现且需要测试**:
- `compatibilitySchema.customResourceDefinition` - CRD schema定义
- `compatibilitySchema.requiredVersions.defaultSelection` - 版本选择
- `customResourceDefinitionSchemaValidation.action` - Deny/Warn动作
- Webhook validation - 验证CRD变更
- Status reporting - 报告兼容性状态

❌ **未实现（不测试）**:
- `compatibilitySchema.ExcludedFields` - 排除字段
- `compatibilitySchema.RequiredVersions.additionalVersions` - 额外版本
- `objectSchemaValidation` - 对象验证
- **CRD Finalizer** - CRD删除保护（已发现未实现）

### 核心验证点

根据PR要求，需要验证以下4个核心场景：

1. ✅ **Create compatibility requirement matching existing schema**
   - 创建匹配现有schema的CompatibilityRequirement

2. ✅ **Observe status of compatibility schema**
   - 观察兼容性状态

3. ✅ **Try to remove a field from the schema**
   - 尝试从schema中移除字段

4. ✅ **Should be rejected as this would break the controller**
   - 应该被拒绝因为这会破坏controller

## 🧪 测试用例

### 第一部分：基础功能与状态验证

#### TC-01: 创建匹配现有 Schema 的 CompatibilityRequirement

**目标**: 验证Operator能够正确处理与现有CRD完全匹配的CompatibilityRequirement

**对应核心验证点**: #1 - Create compatibility requirement matching existing schema

**前置条件**:
- 集群中存在CAPI相关CRD（如 `machinesets.cluster.x-k8s.io`）
- Compatibility Requirements Operator已安装并运行

**测试步骤**:
1. 获取现有CRD的完整schema
   ```bash
   oc get crd machinesets.cluster.x-k8s.io -o yaml
   ```

2. 创建CompatibilityRequirement，将上述schema嵌入到 `spec.compatibilitySchema.customResourceDefinition.data`
   ```yaml
   apiVersion: apiextensions.openshift.io/v1alpha1
   kind: CompatibilityRequirement
   metadata:
     name: test-cr-tc01
   spec:
     compatibilitySchema:
       customResourceDefinition:
         name: machinesets.cluster.x-k8s.io
         type: YAML
         data: |
           <完整的CRD YAML>
       requiredVersions:
         defaultSelection: StorageOnly
       customResourceDefinitionSchemaValidation:
         action: Deny
   ```

3. 应用CompatibilityRequirement
   ```bash
   oc apply -f test-cr-tc01.yaml
   ```

4. 等待reconciliation（约10秒）

5. 检查status conditions
   ```bash
   oc get compatibilityrequirement test-cr-tc01 -o jsonpath='{.status.conditions}' | jq
   ```

**预期结果**:
- ✅ CompatibilityRequirement创建成功
- ✅ `status.conditions` 包含:
  - `Admitted: True` - CR被接受
  - `Compatible: True` - Schema兼容
  - `Progressing: False` - 处理完成
- ✅ 没有错误条件

**失败场景**:
- ❌ `Admitted: False` - 可能是schema格式错误
- ❌ `Compatible: False` - CRD已被修改，不再匹配

**验证命令**:
```bash
./test-manifests/comprehensive-test.sh
# 或单独运行 TC-01
```

---

#### TC-02: 观察无效CRD的Status流转

**目标**: 验证当引用的CRD不存在时，Status能正确反映错误

**对应核心验证点**: #2 - Observe status of compatibility schema

**前置条件**:
- Operator正常运行

**测试步骤**:
1. 创建引用不存在CRD的CompatibilityRequirement
   ```yaml
   spec:
     compatibilitySchema:
       customResourceDefinition:
         name: invalid-crd-does-not-exist.test.io
         # ... schema定义
   ```

2. 应用CR

3. 观察status变化

**预期结果**:
- ✅ CR创建成功（API层不校验CRD是否存在）
- ✅ `status.conditions` 包含:
  - `Admitted: False` 或
  - `Compatible: False/Unknown`
  - 错误信息说明CRD不存在

**验证点**:
- Status能准确反映配置问题
- 错误信息清晰易懂

---

### 第二部分：破坏性变更防护测试 (Webhook Validation)

#### TC-03: 尝试移除字段 - 应该被拒绝 ⭐

**目标**: 验证Webhook能够阻止删除字段的破坏性操作

**对应核心验证点**:
- #3 - Try to remove a field from the schema
- #4 - Should be rejected as this would break the controller

**这是最重要的测试用例**，验证PR的核心功能。

**背景**:
- OpenShift controller依赖CRD中的某些字段
- 如果外部组件（如HyperShift）升级CRD并删除这些字段
- 会导致OpenShift controller失败
- Webhook必须阻止这种破坏性修改

**前置条件**:
1. 已创建CompatibilityRequirement引用目标CRD
2. CompatibilityRequirement的status为Admitted=True
3. Webhook已注册并运行

**测试步骤**:

1. 创建CompatibilityRequirement（包含完整schema）
   ```bash
   oc apply -f test-cr-tc03.yaml
   ```

2. 等待Admitted=True

3. 备份当前CRD
   ```bash
   oc get crd machinesets.cluster.x-k8s.io -o yaml > backup.yaml
   ```

4. 修改CRD，删除一个字段（如 `spec.replicas`）
   ```bash
   yq eval 'del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.replicas)' backup.yaml > modified.yaml
   ```

5. 尝试应用修改（使用dry-run测试）
   ```bash
   oc apply -f modified.yaml --dry-run=server
   ```

**预期结果**:
- ✅ Webhook **拒绝**修改
- ✅ 错误信息包含 "compatibility" 或 "would break" 等关键词
- ✅ CRD保持不变

**错误输出示例**:
```
Error from server: admission webhook "..." denied the request:
removing field 'spec.replicas' would break compatibility requirement test-cr-tc03
```

**失败场景**:
- ❌ 修改被允许 - Webhook未正确工作
- ❌ 修改被拒绝但原因不是Webhook - 可能是其他admission controller

**验证命令**:
```bash
# 使用comprehensive-test.sh中的TC-03
./test-manifests/comprehensive-test.sh

# 或手动测试
./test-manifests/TEST_SCENARIO_FIELD_REMOVAL.md
```

**注意事项**:
- ⚠️ 不要在生产环境删除真实CRD的字段
- ✅ 使用 `--dry-run=server` 进行安全测试
- ✅ 始终备份CRD

---

#### TC-04: 添加字段 - 应该被允许

**目标**: 验证Webhook允许添加新字段（向后兼容的修改）

**对应核心验证点**: Schema loosening应该被允许

**背景**:
- 添加新的可选字段不会破坏现有controller
- OpenShift会忽略（prune）不认识的字段
- 这种修改应该被允许

**前置条件**:
- CompatibilityRequirement已创建且Admitted=True

**测试步骤**:

1. 修改CRD，添加一个新字段
   ```bash
   yq eval '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.newField = {"type": "string"}' original.yaml > with-new-field.yaml
   ```

2. 尝试应用
   ```bash
   oc apply -f with-new-field.yaml --dry-run=server
   ```

**预期结果**:
- ✅ Webhook **允许**修改
- ✅ 命令执行成功，无错误
- ✅ 如果实际应用，CRD会更新成功

**失败场景**:
- ❌ Webhook拒绝添加字段 - 规则过于严格

---

### 第三部分：边界与异常测试

#### TC-05: 无效的Schema格式

**目标**: 验证API或Controller能够处理格式错误的schema

**测试步骤**:
1. 创建包含无效YAML的CompatibilityRequirement
   ```yaml
   spec:
     compatibilitySchema:
       customResourceDefinition:
         data: |
           invalid yaml: {{{
   ```

**预期结果**:
- API层拒绝创建，或
- Status中报告格式错误

---

#### TC-06: Webhook故障保护策略 ⭐

**目标**: 验证Webhook配置了正确的失败策略

**重要性**:
- 如果Webhook失败且配置为`failurePolicy: Ignore`
- 破坏性的CRD修改会被允许
- 这是安全风险

**测试步骤**:

1. 检查ValidatingWebhookConfiguration
   ```bash
   oc get validatingwebhookconfiguration \
     openshift-compatibility-requirements-apiextensions-k8s-io-v1-customresourcedefinition-validation \
     -o yaml
   ```

2. 查看failurePolicy
   ```yaml
   webhooks:
   - failurePolicy: Fail  # 或 Ignore
   ```

**预期结果**:
- ✅ `failurePolicy: Fail` (fail-closed)
  - Webhook不可用时，CRD修改被拒绝
  - **这是安全的配置**

**失败场景**:
- ❌ `failurePolicy: Ignore` (fail-open)
  - Webhook不可用时，CRD修改被允许
  - **这是不安全的**

**额外验证**:
- Webhook Service存在
- Service有可用的Endpoints
- Pod健康状态正常

---

## 🚀 执行测试

### 快速开始

```bash
cd /Users/zhsun/go/src/github.com/openshift/cluster-capi-operator/test-manifests

# 确保有权限
chmod +x comprehensive-test.sh

# 运行所有测试
./comprehensive-test.sh
```

### 单独运行测试

每个测试函数可以单独调用：
```bash
# 只运行TC-03（最重要的测试）
source comprehensive-test.sh
check_prerequisites
tc03_remove_field_rejected
```

### 测试输出

脚本会输出：
- ✅ 每个测试的Pass/Fail状态
- 📊 最终统计信息
- 🔍 详细的错误信息（如果有）

示例输出：
```
================================================
测试总结 (Test Summary)
================================================

总测试数 (Total):   6
通过 (Passed):      5
失败 (Failed):      1
跳过 (Skipped):     0

✗ 有测试失败，请检查上述输出
```

## 📝 已知问题与限制

### 1. CRD Finalizer未实现 ⚠️

**问题**: PR #459的代码中没有实现给目标CRD添加finalizer的功能

**影响**:
- ✅ Webhook validation工作正常
- ❌ CRD删除保护不工作
- 即使CompatibilityRequirement存在且Admitted=True
- 用户仍然可以删除目标CRD

**详细分析**: 参见 [BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md](BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md)

**测试影响**:
- 本测试计划**不包含**finalizer相关测试
- 如果需要测试finalizer，需要等待功能实现

### 2. TC-03的局限性

**问题**: TC-03使用dry-run模式测试

**原因**:
- 不能在真实环境中修改生产CRD
- 需要安全的测试方法

**局限**:
- 只能验证admission控制层
- 不能验证实际应用后的效果

**建议**:
- 在非生产环境进行完整测试
- 或使用测试CRD进行真实修改测试

### 3. Webhook时序问题

**问题**: Webhook可能需要时间生效

**解决**:
- 创建CompatibilityRequirement后等待足够时间
- 脚本中已加入`sleep 10`
- 如果测试失败，尝试增加等待时间

## 🔗 相关文档

- **PR**: https://github.com/openshift/cluster-capi-operator/pull/459
- **EP**: https://github.com/openshift/enhancements/blob/master/enhancements/cluster-api/crd-compatibility-checker.md
- **测试脚本**: [comprehensive-test.sh](test-manifests/comprehensive-test.sh)
- **Bug报告**: [BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md](BUG_REPORT_CRD_FINALIZER_NOT_IMPLEMENTED.md)
- **诊断工具**: [diagnose-finalizer.sh](test-manifests/diagnose-finalizer.sh)

## ✅ 测试通过标准

PR #459可以被认为测试通过，当满足以下条件：

### 必须通过 (Critical)
- ✅ TC-01: 创建匹配schema的CR - Status正确
- ✅ TC-03: 删除字段被Webhook拒绝 ⭐⭐⭐
- ✅ TC-06: Webhook配置为fail-closed

### 应该通过 (Important)
- ✅ TC-02: 无效CRD的Status正确
- ✅ TC-04: 添加字段被允许
- ✅ TC-05: 无效schema被处理

### 可以跳过 (Optional)
- 如果某些CRD不存在，相关测试可以跳过

### 已知不通过（预期）
- ❌ CRD Finalizer测试 - 功能未实现

## 📊 测试矩阵

| 测试用例 | 功能 | 优先级 | 自动化 | 状态 |
|---------|------|--------|--------|------|
| TC-01 | 创建匹配schema的CR | High | ✅ | Ready |
| TC-02 | Status观察 | Medium | ✅ | Ready |
| TC-03 | 删除字段被拒绝 | **Critical** | ✅ | Ready |
| TC-04 | 添加字段被允许 | High | ✅ | Ready |
| TC-05 | 无效schema处理 | Low | ✅ | Ready |
| TC-06 | Webhook故障策略 | High | ✅ | Ready |

## 🎯 总结

本测试计划覆盖了PR #459实现的核心功能：

1. ✅ **Webhook Validation** - 防止破坏性CRD修改
2. ✅ **Status Reporting** - 准确报告兼容性状态
3. ✅ **基础功能** - CompatibilityRequirement的创建和管理

**未覆盖**（因为未实现）:
- ❌ CRD Finalizer
- ❌ ExcludedFields
- ❌ Additional RequiredVersions
- ❌ Object Schema Validation

**下一步**:
1. 运行comprehensive-test.sh
2. 如果TC-03失败，在PR中报告
3. 将测试结果反馈到PR

---

**文档版本**: 1.0
**创建日期**: 2026-02-26
**适用PR**: #459
**测试环境**: OpenShift 4.22+
