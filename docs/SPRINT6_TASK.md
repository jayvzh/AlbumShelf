# Sprint 6 开发任务书（AI IDE Task Specification）

> 用法：将本文档整体复制给 AI IDE（Cursor / Claude Code / Windsurf 等）作为 Sprint 6 的任务输入。
> Sprint 6 定位：**Docker 部署收尾（Sprint 7）前的功能收口 Sprint**——完成双页 Spread 全部功能，补齐 PRD 尚未覆盖的最后缺口（旋转、favorites 表预留），使项目达到「功能完整、可交付部署」状态。

---

## 1. 任务上下文

在开始编码前，AI 必须先完整阅读以下文档：

```
docs/PRD.md（F002 旋转 / F007 双页 / F010 favorites 预留 / §7 成功标准）
docs/PROJECT_STRUCTURE.md（§12 组件职责、§13 Store 设计、§16 Flow 2/3）
docs/SPREAD_ENGINE.md（本 Sprint 核心依据：§3 算法与渲染模型、§6 单测清单）
docs/API.md（§3.6 folder settings、§4 types/spread.ts 契约）
docs/UI_DESIGN.md（§1 PageMode 入口、§3 SpreadControls、§4 状态设计、§5 快捷键、§6 相邻 1 帧预加载）
docs/DATA_MODEL.md（§2.2 folder_settings 列、§2.3 favorites 表）
docs/SPRINT_PLAN.md（Sprint 6 部分）
docs/DEVELOPMENT_RULES.md
开发规则.md（根目录精简版：命令与自测约定）
```

### 现状基线（动手前必须先读对应代码再改）

| 现状 | 说明 |
| --- | --- |
| 占位文件已存在 | `frontend/src/utils/spread.ts`、`frontend/src/types/spread.ts`、`frontend/src/stores/spread.ts`、`frontend/src/components/viewer/SpreadFrame.vue`、`frontend/src/components/viewer/SpreadControls.vue`（仅含 `TODO(Sprint 6)` 注释） |
| viewer store | `isOpen/images/currentIndex/mode`（mode 恒 `'single'`）`/variant`；`next()/previous()/select()` 均按**图**推进 |
| settings store / types | 仅 sort_mode/sort_direction/regex_pattern/regex_config 四字段，全 nullable；`save()` 仅 regex 模式携带 regex 字段 |
| 后端 folder_settings | 表列已建好（`page_mode/read_order/wide_ratio/single_first_page/single_last_page/view_mode`），但 model/request/service/repository/handler 全链路**尚未接入**任何 spread 字段；`settings_handler_test.go` fixture 已按 API.md 契约发送 `view_mode`（实现一直忽略） |
| favorites 表 | 未建（PRD F010：MVP 只预留数据表） |
| 前端测试基建 | 无 Vitest（package.json 无 test 脚本） |
| Viewer 渲染 | `ImageViewer.vue` 在 `ViewerCanvas.vue` 内渲染**单张 img**；`preloadAdjacent()` 只预载相邻 1 张预览图 |
| 快捷键 | `useKeyboard.ts` 有 ArrowLeft/ArrowRight/Space/KeyF/Digit0/Digit1/Escape，**无旋转键**；`useViewer.ts` 无 rotation |
| 入口 | `AppHeader.vue` 仅 Logo + 路径 + SortMenu，无 PageMode 切换；`ViewerToolbar.vue` 页码按图显示 `index+1 / total` |

---

## 2. 本 Sprint 目标

**双页 Spread（自动拼页）+ 功能收口**

1. `utils/spread.ts`：SpreadEngine 纯函数 + 单元测试（用例见 SPREAD_ENGINE.md §6，共 7 项）
2. spread store（5 个选项 + frames + 映射）与 ViewerStore 帧化（翻页按帧推进）
3. `SpreadFrame.vue`（单画布帧渲染）、`SpreadControls.vue`（四项控制）
4. folder_settings 5 个 spread 字段全链路接入（后端 + 前端 + 保存重开生效）
5. Filmstrip 帧级联动（当前帧所有图片高亮、点击缩略图定位所在帧、切换模式不跳变）
6. 补齐 PRD F002「旋转」（本 Sprint 增量，SPRINT_PLAN Sprint 6 范围）
7. favorites 表建表预留（PRD F010：MVP 只建表不实现功能）
8. 文档同步：API.md §3.6 view_mode 标注、README 修正、SPRINT_PLAN 回写

---

## 3. 允许创建 / 填充 / 修改的文件（精确清单）

> Sprint 6 多数文件已存在，分三类管理。**超出此清单的任何创建/修改一律禁止。**

### A. 新建（仅 1 个）

```
frontend/src/utils/spread.test.ts        # Vitest 单元测试（SPREAD_ENGINE.md §6 七项用例）
```

### B. 填充占位（5 个，现为 TODO(Sprint 6) 注释）

```
frontend/src/utils/spread.ts             # SpreadEngine 纯函数
frontend/src/types/spread.ts             # 按 API.md §4 契约实现
frontend/src/stores/spread.ts            # SpreadStore（PROJECT_STRUCTURE.md §13）
frontend/src/components/viewer/SpreadFrame.vue
frontend/src/components/viewer/SpreadControls.vue
```

### C. 允许修改的既有文件（附理由，改动保持最小）

```
frontend/
├── src/
│   ├── stores/
│   │   ├── viewer.ts                    # 帧化：currentFrameIndex、翻页按帧、select(imageIndex) 经映射定位
│   │   └── settings.ts                  # 载入/保存 5 个 spread 字段（null→默认值）
│   ├── types/
│   │   └── settings.ts                  # FolderSettings 增加 5 个 spread 字段（nullable）
│   ├── services/
│   │   └── settings.service.ts          # 请求/响应序列化携带 spread 字段
│   ├── composables/
│   │   ├── useViewer.ts                 # 增加 rotation 状态与 rotate()（UI Runtime State）
│   │   └── useKeyboard.ts               # 增加 KeyR → rotate
│   ├── components/
│   │   ├── viewer/
│   │   │   ├── ImageViewer.vue          # 渲染 SpreadFrame、帧内全部图片加载完成后再 fit、相邻 1 帧预载、rotation 联动
│   │   │   ├── ViewerCanvas.vue         # 槽位适配帧容器（Transform 仍作用于整帧）
│   │   │   └── ViewerToolbar.vue        # 页码按帧显示（Spread 模式）、旋转按钮
│   │   ├── filmstrip/
│   │   │   ├── Filmstrip.vue            # 帧级高亮集合、点击传 imageIndex
│   │   │   └── ThumbnailItem.vue        # 支持多选高亮态（当前帧内）
│   │   └── layout/
│   │       └── AppHeader.vue            # PageMode（单页/双页）切换入口（UI_DESIGN.md §1）
│   ├── composables/useFilmstrip.ts      # 自动居中以帧为单位（复用现有 scrollToCenter）
│   ├── package.json                     # devDependencies 增加 vitest；scripts 增加 "test": "vitest run"
│   └── vite.config.ts                   # test 配置（environment: 'node'）
backend/
├── internal/
│   ├── repository/
│   │   ├── database.go                  # 仅追加 CREATE TABLE IF NOT EXISTS favorites（DATA_MODEL.md §2.3）
│   │   └── settings_repository.go       # 5 个 spread 列读写（列已存在，无需迁移）
│   ├── model/settings.go                # FolderSettings 增加 5 字段
│   ├── api/
│   │   ├── request/settings_request.go  # 接收 5 字段（可空）
│   │   └── handler/
│   │       ├── settings_handler.go      # 响应输出 5 字段；view_mode 保持不输出
│   │       └── settings_handler_test.go # 更新 fixture 断言（spread 字段回显；view_mode 相关断言清理）
│   └── service/settings_service.go      # 规范化非法值 + 空缺默认值
docs/
├── API.md                               # §3.6 view_mode 标注为保留字段（示例值改 null + 表格下加一行说明）
├── SPRINT_PLAN.md                       # Sprint 6 任务清单补记：旋转、favorites 表、文档同步（完成范围回写）
README.md                                # 第 4 行「双图对比」修正为双页阅读（NeeView 式自动拼页）
```

---

## 4. 禁止开发（仅产品边界，本 Sprint 保持最小约束）

本 Sprint **禁止**创建或实现：

```
Dummy Page 补位、静态奇偶配对（staticWidePage）      （Phase 2，SPREAD_ENGINE.md §5）
DividePage 横图拆屏 / Panorama 连续滚动              （Phase 2 / 远期）
任何图片对比工具功能                                  （已从产品范围移除）
view_mode 字段的任何功能接入                          （保留字段，见 §5.7；前后端不读写、UI 不暴露）
SettingsPage 页面 / AppHeader 独立 Settings 入口     （PRD/API 无此需求；spread 设置入口 = PageMode 切换 + SpreadControls）
favorite_repository.go 及任何收藏 UI/API            （MVP 只建表，PRD F010）
第 5 个 Pinia Store / AppStore                       （PROJECT_STRUCTURE.md §13 仅允许 4 个）
Docker 迭代与部署验证                                （Sprint 7；禁止为验证功能反复 docker compose up --build）
```

其余说明：

- 除 `vitest`（devDependency）外**禁止引入任何新依赖**；Headless UI 已有则复用
- 页间距 MVP 用**固定常量**（如 8px）+ `// TODO(Phase 2): 页间距可配置（PRD F007）` 注释扩展点，不做配置项
- 架构红线（分层、数据流、禁止 Component 直接 fetch 等）以 docs/DEVELOPMENT_RULES.md 与 PROJECT_STRUCTURE.md §19 为准，本文档不重复

---

## 5. 技术要求

### 5.1 SpreadEngine（utils/spread.ts，纯函数）

严格按 SPREAD_ENGINE.md §3.3 伪代码实现：

- `isLandscape(width, height, wideRatio)`：`width > height × wideRatio`；`width/height` 为 null（解析失败）时**按竖图处理**（SPREAD_ENGINE.md §4）
- 竖图两两拼帧；横图独占一帧；下一张是横图时当前竖图独占；末尾落单独占
- 封面：`i === 0 && singleFirstPage` 时首张竖图不拼页；封底：按伪代码 `i+1 === last && singleLastPage` 分支（开启后最后两张不拼页）
- `readOrder === 'right_to_left'` 时**帧内两图顺序反转**（第一页显示在右侧）
- `pageMode === 'single'` 时每张图各占一帧（单页模式是「全单图帧」的退化情形，翻页/联动逻辑统一）
- 输出：`frames: SpreadFrame[]`（`type: 'single'|'spread'`，1 张图为 single、2 张图为 spread）+ `imageToFrame: number[]`（imageIndex → frameIndex 映射）
- 函数签名与类型对齐 API.md §4 `types/spread.ts` 契约；引擎不做 DOM/网络操作，可被单测直接调用

### 5.2 spread store 与 ViewerStore 帧化

- SpreadStore 状态严格对齐 PROJECT_STRUCTURE.md §13：`pageMode/wideRatio/readOrder/singleFirstPage/singleLastPage/frames/imageToFrame`；actions：`buildFrames()/togglePageMode()/setReadOrder()`，同族 setter（`setWideRatio()` 等）按同一模式补齐——任何选项变化后必须重建 frames
- 选项默认值（与 §5.7 后端默认一致）：`pageMode 'single'`、`wideRatio 1.0`、`readOrder 'left_to_right'`、`singleFirstPage true`、`singleLastPage false`
- 打开目录 / images 变化 / 选项变化时 `buildFrames()` 重建；buildFrames 的输入来自 FolderStore 的 images
- ViewerStore 帧化：
  - 新增 `currentFrameIndex`；`next()/previous()` 移动一帧，**不循环**（末尾/开头停住，与现有行为一致）
  - `select(imageIndex)`（Filmstrip 点击）经 `imageToFrame` 映射定位到所在帧
  - `currentFrame` computed = `frames[currentFrameIndex]`；`currentImage` = 帧内图片（供 ImageInfo / variant / 预载使用）
  - `mode` 由恒 `'single'` 改为跟随 spread store 的 pageMode
- **模式切换不跳变**（SPREAD_ENGINE.md §3.5）：切换前记录当前帧内**焦点图片的 imageIndex**，重建 frames 后用映射反查新 frameIndex 写回，不允许保留旧帧号
- 键盘链路不变：`useKeyboard → Viewer Action → ViewerStore`（PROJECT_STRUCTURE.md §14）

### 5.3 SpreadFrame.vue（单画布帧渲染，SPREAD_ENGINE.md §3.4）

- props：`frame: SpreadFrame`、`readOrder`；渲染 1~2 张 `<img>`（preview 变体 URL 统一走 `image.service.ts` 构造）
- 等高对齐：帧内两图按高度统一缩放、垂直居中（NeeView WidePageStretch=UniformHeight）；`right_to_left` 时视觉左右顺序反转
- 页间距固定常量 + Phase 2 注释扩展点（见 §4）
- 单页帧 = 只含一张图的退化情形，渲染逻辑统一（不做分支重复实现）

### 5.4 ViewerCanvas / ImageViewer / ViewerToolbar 改造

- ViewerCanvas：Transform（scale/translate）作用于**整个帧容器**，SpreadFrame 作为其唯一内容槽；Zoom/Pan/Fit/100%/Fullscreen 全部复用现有 useViewer 逻辑
- ImageViewer：
  - 加载当前帧内全部图片（1~2 张）；**帧内全部图片 onload 后**计算整帧内容尺寸上报 `setContentSize()` 再 `fit()`（等高对齐下：以两图 natural 高归一化，帧宽 = 两图归一化宽之和 + 页间距，帧高 = 归一化高；单图帧退化为该图 natural 尺寸）
  - loading 覆盖逻辑按「帧」触发（禁止白屏闪烁）
  - `preloadAdjacent()` 改为预载**相邻 1 帧的预览图**（spread 帧内可能 2 张，UI_DESIGN.md §6）
- ViewerToolbar：
  - Spread 模式页码按帧显示：`帧 {frameIndex+1} / {framesTotal}`；Single 模式保持图级显示
  - 新增旋转按钮（见 §5.5）

### 5.5 旋转（PRD F002，本 Sprint 增量）

- `useViewer` 增加 `rotation`（0/90/180/270）与 `rotate()`（每次 +90°），属 UI Runtime State **不进 Store**（UI_DESIGN.md §4）
- Transform 追加 rotate，**作用于整帧**（与 Zoom/Pan 同层）
- 入口：ViewerToolbar 旋转按钮 + 快捷键 `R`（useKeyboard 的 KEY_ACTIONS 与 KeyboardHandlers 增加 rotate）
- 旋转 90/270 后需以旋转后的内容尺寸重新适配（rotate 后自动 fit）；切换帧时 rotation 重置为 0（与换图 variant 回落 preview 同理）

### 5.6 Filmstrip 帧级联动

- 高亮：当前帧包含的**所有图片**缩略图同时高亮（高亮集合 = 帧内 imageIndex 集合）
- 点击缩略图 → `select(imageIndex)` → 映射定位所在帧（Flow 2）
- 自动居中：帧变化后以帧内**首图索引**复用现有 `scrollToCenter`（useFilmstrip 布局数学不改，只改入参语义）

### 5.7 folder_settings 5 字段全链路（含 view_mode 决策）

**决策（已确认）：`view_mode` 为早期遗留字段，本 Sprint 不接入。** 后端不读写该列、响应不输出该字段、前端类型不含该字段；docs/API.md §3.6 示例中 `view_mode` 改为 `null`，并在字段说明表下加一行：「`view_mode`：保留字段，MVP 不启用，后端不读写，响应恒为 null」。

其余 5 字段接入要求：

- 后端（列已存在，无需迁移）：
  - request：5 字段全部可空
  - service 规范化：非法值与空缺一律回退默认——`page_mode` ∉ {single, spread} → `single`；`read_order` 非法 → `left_to_right`；`wide_ratio` 缺失或 ≤ 0 → `1.0`；`single_first_page` 缺失 → `true`；`single_last_page` 缺失 → `false`
  - handler 响应输出 5 字段（GET/PUT 行为与现有 sort/regex 字段风格一致）
- 前端：
  - `types/settings.ts` FolderSettings 增加 5 个 nullable 字段；`settings.service.ts` 序列化
  - settings store：load 后 null → 默认值（同上），并同步进 spread store；SpreadControls 任何修改**即时生效**（重建 frames）并按现有设置保存机制自动 `PUT`
  - 保存后刷新页面 / 重开目录 → 设置仍生效（对齐 Sprint 4「每目录记忆」体验）
- 测试：`settings_handler_test.go` 更新——spread 字段保存后回显正确；`view_mode` 相关 fixture 断言清理

### 5.8 favorites 表预留（PRD F010）

- 仅在 `database.go` 追加 `CREATE TABLE IF NOT EXISTS favorites`（列定义照 DATA_MODEL.md §2.3：`id INTEGER PRIMARY KEY, file_path TEXT UNIQUE, created_at DATETIME`）
- 不建 repository、不写任何读写代码

### 5.9 Vitest 测试基建（本 Sprint 新引入，需说明理由）

- `package.json`：devDependencies 增加 `vitest`，scripts 增加 `"test": "vitest run"`
- `vite.config.ts`：`test: { environment: 'node' }`（只测纯函数，无需 jsdom）
- `spread.test.ts` 必须覆盖 SPREAD_ENGINE.md §6 全部 7 项：
  1. 全竖图两两配对 `[1,2],[3,4],[5,6]`
  2. 横图打断：竖,竖,横,竖,竖 → `[1,2],[3],[4,5]`
  3. 末尾落单竖图独占一帧
  4. 封面单页开启：第一张竖图不与第二张拼页
  5. WideRatio=1.2 时正方形判为竖图；WideRatio=0.8 时判为横图
  6. right_to_left：双页帧内图片顺序反转
  7. imageIndex→frameIndex 映射正确；切换模式后当前图片保持不跳变

### 5.10 文档同步（本 Sprint 交付的一部分）

- `docs/API.md` §3.6：view_mode 标注（见 §5.7）
- `README.md` 第 4 行：「双图对比」→「双页阅读（NeeView 式自动拼页）」
- `docs/SPRINT_PLAN.md` Sprint 6 任务清单补记：旋转补齐、favorites 表预留、文档同步三项增量

---

## 6. 验收标准

**开发模式验证（默认方式）：`./scripts/dev.sh start`（前端 :5160 / 后端 :8080）**

场景 E（PRD 双页阅读）：

- 含竖图、横图混合的目录：竖图两两拼页、横图自动独占整屏、按帧混合翻页，`←`/`→`/`Space` 一次翻一帧
- `right_to_left` 模式下双页左右顺序正确（日漫右开本）
- 封面单页开启时第一张竖图不拼页；关闭后恢复拼页
- 切换 Single/Spread 当前图片不跳变；点击 Filmstrip 缩略图直达所在帧；当前帧所有图片同时高亮并自动居中
- Zoom/Pan/Fit/100% 作用于整帧，双页等高对齐、垂直居中

功能收口增量：

- 旋转：工具栏按钮与 `R` 键生效，旋转作用于整帧，旋转后 Fit 正确，换帧重置
- folder_settings：修改 pageMode/readOrder/wideRatio/封面封底开关 → 保存 → 刷新/重开目录后仍生效；非法值（手改 DB 或请求）回退默认
- Single 模式回归：Sprint 2-5 全部行为不回归（按图翻页、加载原图切换、排序、Regex、Filmstrip 虚拟滚动）

命令与测试：

```text
[后端] go build ./... 与 go test ./... 通过
[前端] pnpm build 通过
[前端] pnpm test 通过（spread.test.ts 七项用例全绿）
```

---

## 7. AI 自检清单（完成后必须逐项确认）

```text
[ ] go build ./... / go test ./... / pnpm build / pnpm test 全部通过
[ ] spread.test.ts 覆盖 SPREAD_ENGINE.md §6 全部 7 项且全绿
[ ] 开发模式场景 E 全流程验证通过（含 right_to_left、封面单页、模式切换不跳变）
[ ] 翻页/预载/Filmstrip 高亮均以帧为单位；Single 模式无回归
[ ] 旋转（按钮 + R 键）作用于整帧，换帧重置
[ ] folder_settings 5 字段：保存 → 刷新仍生效；非法值回退默认（wideRatio 1.0 / left_to_right / true / false / single）
[ ] view_mode 前后端均未读写、UI 未暴露；API.md 已标注保留字段（示例为 null）
[ ] favorites 表已建（CREATE TABLE IF NOT EXISTS），且无任何收藏功能代码
[ ] 创建/修改的文件与 §3 清单完全一致；未引入 vitest 之外的新依赖
[ ] 未触碰 §4 禁止清单（Dummy Page / 静态配对 / DividePage / Panorama / 对比工具 / SettingsPage / 第 5 个 Store / Docker 迭代）
[ ] README.md 第 4 行已修正；SPRINT_PLAN.md Sprint 6 已回写增量范围
[ ] 页间距为固定常量并带 Phase 2 扩展点注释
```

---

## 8. 交付物

- 满足 §3 清单的全部代码与文档修改
- 自检结果汇报（逐项打勾）
- 开发模式验证记录：场景 E 操作说明与结果（含 right_to_left 截图或命令输出）、`pnpm test` 输出、`go test` 输出
