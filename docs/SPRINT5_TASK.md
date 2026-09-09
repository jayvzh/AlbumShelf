# Sprint 5 开发任务书（AI IDE Task Specification）

> 用法：将本文档整体复制给 AI IDE 作为 Sprint 5 的任务输入。
> 模板与 SPRINT0_TASK.md 一致：任务上下文 / 目标 / 允许创建的文件 / 禁止项 / 技术要求 / 验收标准 / 自检清单。

---

## 1. 任务上下文

在开始编码前，AI 必须先完整阅读以下文档：

```
docs/PRD.md（F005 / F006 / 场景 D）
docs/ARCHITECTURE.md
docs/PROJECT_STRUCTURE.md
docs/DATA_MODEL.md（§2.2 folder_settings）
docs/API.md（§3.2 / §3.6 / §3.7 / §4 / 错误码表）
docs/SORT_ENGINE.md（本 Sprint 的核心算法文档）
docs/UI_DESIGN.md（§1 / §6 Regex Editor）
docs/DEVELOPMENT_RULES.md
docs/SPRINT_PLAN.md（本 Sprint 部分）
开发规则.md（根目录精简版：命令与自测约定）
```

**代码基线（Sprint 0-4 已完成并验收）**：

- `internal/sorting/`：sorter.go（Engine 分发 + 稳定排序）已有 filename/natural/time/size 四个 Sorter，`Sorter.Sort([]model.Image, model.SortOptions) []model.Image`；regex_sorter.go / rule_sorter.go 为 `TODO(Sprint 5)` 占位
- `model.SortOptions` 仅有 Mode/Direction；`model/sort.go` 未定义 `regex` 常量与 `SortRule`
- folder_settings 表已含 `regex_pattern` / `regex_config` 列（database.go ensureSchema），但 model / repository / handler / service 均未读写
- 前端 `components/sorting/` 下 RegexEditor.vue / SortRuleEditor.vue 为占位；SortMenu.vue 已有 5 种模式 + 方向 + 保存；`types/sort.ts` 已含 `regex` 联合类型成员

## 2. 本 Sprint 目标

**Regex / 多规则排序**（项目核心特色功能）：

1. `regex_sorter.go` + `rule_sorter.go`（多规则链式比较）
2. `POST /api/v1/sort/preview`（RegexEditor 实时预览）
3. folder_settings 接入 `regex_pattern` + `regex_config`（JSON）
4. 前端 RegexEditor（输入校验 + 实时 Preview 三列展示）+ SortRuleEditor
5. 未匹配文件固定"排最后、保持原序"

## 3. 允许创建 / 修改的文件（精确清单）

> **超出此清单的文件一律禁止创建。** 占位文件只允许填充本 Sprint 内容，其余占位文件保持原样。

**允许创建（新增）**：

```
backend/internal/sorting/regex_sorter_test.go
backend/internal/sorting/rule_sorter_test.go
（测试也可并入既有 sorter_test.go，二选一）
```

**允许填充的占位文件（已预创建）**：

```
backend/internal/sorting/regex_sorter.go
backend/internal/sorting/rule_sorter.go
frontend/src/components/sorting/RegexEditor.vue
frontend/src/components/sorting/SortRuleEditor.vue
```

**允许修改的既有文件**：

```
Backend:
backend/internal/model/sort.go                  # regex 常量、SortOptions.Regex/Rules、SortRule 类型
backend/internal/model/settings.go              # FolderSettings 增加 RegexPattern / RegexConfig
backend/internal/sorting/sorter.go              # Engine 分发 regex 分支；错误传导（见 §5.1）
backend/internal/api/request/folder_request.go  # regex / regex_rules query 参数
backend/internal/api/request/settings_request.go# regex_pattern / regex_config 字段
backend/internal/api/handler/folder_handler.go  # 透传 + INVALID_REGEX 错误映射
backend/internal/api/handler/settings_handler.go# GET/PUT 响应体补 regex 字段
backend/internal/service/folder_service.go      # regex 排序参数与设置回退扩展 + Preview 编排
backend/internal/service/settings_service.go    # validSortMode 支持 regex；保存 regex 字段
backend/internal/repository/settings_repository.go # regex_pattern / regex_config 列读写
backend/internal/api/router.go                  # 注册 POST /api/v1/sort/preview
backend/internal/app/app.go                     # 依赖组装（无新增 Service 类型则可能零改动）
docs/API.md                                     # §3.6 PUT 示例补 regex_config（见 §5.4）

Frontend:
frontend/src/types/sort.ts                      # SortRule / SortPreviewRequest / SortPreviewResponse
frontend/src/types/settings.ts                  # FolderSettings 增加 regex_pattern / regex_config
frontend/src/services/folder.service.ts         # getFolder 增加 regex 参数；新增 previewSort()
frontend/src/stores/folder.ts                   # openFolder opts 透传 regex 参数
frontend/src/stores/settings.ts                 # regex 状态读写与保存
frontend/src/components/sorting/SortMenu.vue    # 菜单加 Regex 项 + 打开 RegexEditor
```

## 4. 禁止开发（未来功能 / 范围外）

```
spread 任何代码：utils/spread.ts、SpreadFrame.vue、SpreadControls.vue、stores/spread.ts（Sprint 6）
folder_settings 的 view_mode / page_mode / read_order / wide_ratio /
  single_first_page / single_last_page 前后端读写（Sprint 6；DB 列已存在，禁止提前接入）
未匹配策略 first / last / ignore 配置化（MVP 固定 last，仅代码注释标注扩展点）
favorite / tag / compare 任何代码（范围外）
前端本地实现任何排序比较逻辑（排序只发生后端；前端仅做 Regex 语法即时提示 + preview 展示）
新增 Store（仍只允许 folder / viewer / spread / settings 四个，本 Sprint 不建 spread）
Docker 迭代（Sprint 7）
```

## 5. 技术要求

### 5.1 Backend（Go 1.24+ / Gin）

**model（model/sort.go、model/settings.go）**

- 新增 `SortModeRegex = "regex"` 常量；`SortOptions` 增加 `Regex string`、`Rules []SortRule`；新增 `SortRule{ Group int; Type string; Direction string }`（Type ∈ number | string，Group 从 1 开始）
- `FolderSettings` 增加 `RegexPattern`、`RegexConfig`（JSON 字符串原样存储）

**sorting 模块（SORT_ENGINE §5/§6）**

- 错误传导：**与 SORT_ENGINE §2 接口定义的偏差**——Regex 编译失败必须能传导为 `INVALID_REGEX`，因此 `Engine.Sort` 需返回 `error`（sorting 包定义 sentinel，如 `ErrInvalidRegex`），同步更新既有调用点与 sorter.go 头部偏差注释
- regex_sorter.go：编译 → 逐文件 `FindStringSubmatch` → 捕获组 → Sort Key → 比较；**稳定排序**（`sort.SliceStable`）；未匹配文件排最后且保持原序
- rule_sorter.go：多规则链式比较（Rule 1 → Rule 2 → ...）；number 按数值、string 按字典序（大小写不敏感，与 natural 一致）；**规则引用的 group 不存在 / group<1 / 非法 type 或 direction → 该规则跳过（不报错）**（SORT_ENGINE §5.5）
- 单规则简写（SORT_ENGINE §5.3）：未提供 rules 时默认单规则 `{ group: 1, type: "number", direction: 顶层 Direction }`；提供了 rules 时以各 rule 自身 direction 为准，顶层 Direction 不再叠加
- 预览纯函数：sorting 包内提供预览入口（如 `RegexPreview`），输入文件名列表，输出 sorted / matches（文件名 → 捕获组值）/ unmatched + `ErrInvalidRegex`；与 RegexSorter 复用"编译 → 捕获 → Sort Key"核心，禁止两套实现

**GET /api/v1/folders 扩展（API.md §3.2）**

- query 参数：`regex`（sort=regex 时必填）、`regex_rules`（URL 编码 JSON，结构与 regex_config 一致：`{"rules":[{...}]}`）
- `regex_rules` JSON 解析失败 → 400 `INVALID_REGEX`
- 设置回退扩展：请求未带 sort 且已保存 `sort_mode=regex` 时，读取 `regex_pattern` + `regex_config`（反序列化为 rules）参与排序；**pattern 为空或编译失败 → 记日志并回退 filename asc，不阻断浏览**（与 Sprint 4 读库失败回退策略一致）

**folder_settings 接入（API.md §3.6 / DATA_MODEL §2.2）**

- GET 响应与 PUT 请求/响应体均增加 `regex_pattern`、`regex_config`（空串序列化为 null，与现有 nilOrString 一致）
- `validSortMode` 支持 `regex`；PUT 保存时两字段原样持久化，读取/使用方容错解析，不做深度校验

**POST /api/v1/sort/preview（API.md §3.7，请求/响应结构以该节为准）**

- 分层：Handler（解析/校验）→ FolderService 新增 Preview 方法（编排）→ sorting 纯函数；**禁止 Handler 内写 Regex 逻辑**
- `mode` 仅支持 `regex`，其他值按 400 `INVALID_REGEX` 处理；预览**不触碰文件系统**（文件名列表来自请求体）
- `matches` / `unmatched` 保持请求原序（unmatched 不出现在 matches 中）；`sorted` 为排序结果
- 错误映射：`ErrInvalidRegex` → 400 `INVALID_REGEX`（folder_handler 与 settings_handler 同款 switch 风格）

### 5.2 Frontend（Vue 3 + TS + Pinia + Tailwind）

**链路约束**：RegexEditor → settings/folder Store → folder.service → API；组件禁止直接 fetch。

- types/sort.ts：补 `SortRule`、`SortPreviewRequest`、`SortPreviewResponse`，**严格对齐 API.md §4，禁止自造结构**
- folder.service.ts：`getFolder` 增加可选 `regex` / `regexRules`（序列化为 URL 编码 JSON `regex_rules`）；新增 `previewSort(req): POST /sort/preview`
- settings store：state 增加 `regexPattern` / `regexRules`；`load()` 读取（容错解析 regex_config）；`save()` 携带两字段
- SortMenu.vue：菜单增加「Regex」项，选中后打开 RegexEditor；「保存到当前目录」把 regex 设置一并落库
- RegexEditor.vue（UI_DESIGN §6）：
  - 复用 `components/common/AppModal.vue` 承载
  - **输入即校验**：JS `new RegExp` 即时红框提示；注意 Go RE2 与 JS 正则存在语法差异（如反向引用、lookahead），**后端 INVALID_REGEX 为权威**，preview 响应报错同样红框提示
  - 规则编辑：SortRuleEditor 行列表（group ≥1 数字输入、type 下拉 number/string、direction 升/降），可增删；不添加规则时按单规则简写预览
  - **实时 Preview 三列展示**（文件名 | Groups 提取值 | 排序后顺序）：300ms 防抖调 previewSort；unmatched 文件单独分组展示在最后
  - 「应用」：写 settings store 本地状态 → `folderStore.openFolder(带 regex 参数)` → 关闭弹窗；regex 为空时禁用应用
- SortRuleEditor.vue：单条规则行编辑组件，v-model 受控，不含请求逻辑

### 5.3 契约与文档同步

- PUT /api/v1/folder/settings 请求体在 API.md §3.6 示例基础上补充 `regex_config` 字段（与 DATA_MODEL §2.2、PRD F006 一致），**交付时回写 docs/API.md**（开发规则 §10：接口有变同步文档）

## 6. 验收标准

**开发模式验证（`./scripts/dev.sh start`，前端 http://localhost:5160）**：

```text
场景 D 全流程：
  准备目录：chapter1_page1.jpg / chapter1_page10.jpg / chapter2_page1.jpg / chapter10_page1.jpg
  （可再加 1-2 个不匹配文件，如 cover.txt 改名的非匹配图或乱名图）
  ↓ SortMenu 选「Regex」→ RegexEditor 输入 chapter(\d+)_page(\d+)
  ↓ 规则：Group 1 number asc、Group 2 number asc
  ↓ Preview 三列正确：chapter10_page2 → Group1=10, Group2=2（示例）
  ↓ 应用后列表顺序：chapter1_page1, chapter1_page10, chapter2_page1, chapter10_page1
     （多规则数字序，非字典序；未匹配文件排最后且保持原序）
  ↓ 「保存到当前目录」→ 刷新页面 / 重新进入目录 → 排序仍生效
非法 Regex（如 `(chapter`）→ 输入即时红框提示 INVALID_REGEX
```

**单元测试（SORT_ENGINE §8，必须全部覆盖）**：

1. Regex 多规则：chapter 系例顺序正确
2. 未匹配文件排最后且保持原序
3. INVALID_REGEX 返回错误
4. DESC 方向反转
5. 稳定性：同 key 输入顺序保持
6. number / string 两种 type 比较行为
7. 规则引用不存在的 group → 跳过不报错

> 禁止为验证功能反复 `docker compose up --build`；Docker 归 Sprint 7。

## 7. AI 自检清单（完成后必须逐项确认）

```text
[ ] go build ./... 通过
[ ] go test ./... 通过（sorting 覆盖 §6 全部 7 项用例）
[ ] pnpm build / pnpm lint 通过
[ ] 开发模式自测通过：场景 D 全流程（预览 → 应用 → 保存 → 重开仍生效）+ 非法 Regex 即时提示
[ ] 创建/修改的文件与 §3 清单完全一致（无多余文件）
[ ] 未创建 §4 禁止清单中的任何模块（尤其 spread 与 page_mode 等字段）
[ ] 排序仅在后端完成；前端无本地排序比较逻辑
[ ] 未匹配文件排最后且保持原序；group 越界规则跳过不报错
[ ] API 行为与 API.md 一致（/folders regex 参数、sort/preview 请求/响应、INVALID_REGEX、folder/settings 新字段）
[ ] docs/API.md 已同步（PUT 示例补 regex_config）
[ ] 未为验证功能反复构建 Docker 镜像
```

## 8. 交付物

- 满足 §3 清单的全部代码文件与测试
- 自检结果汇报（逐项打勾）
- 场景 D 验证输出：preview 接口响应 JSON + 应用排序后的文件顺序（截图或命令输出）
