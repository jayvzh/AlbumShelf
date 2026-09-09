import { defineStore } from 'pinia'
import { getFolderSettings, saveFolderSettings } from '../services/settings.service'
import type { SortDirection, SortMode, SortRule } from '../types/sort'
import type { PageMode, ReadOrder, SpreadOptions } from '../types/spread'

// regex_config 的 JSON 信封（API.md §3.6）
interface RegexConfigEnvelope {
  rules?: SortRule[]
}

// 解析 regex_config 信封字符串；非法 JSON / 结构不符时容错返回空数组
function parseRegexConfig(raw: string | null): SortRule[] {
  if (!raw) return []
  try {
    const env = JSON.parse(raw) as RegexConfigEnvelope
    return Array.isArray(env.rules) ? env.rules : []
  } catch {
    return []
  }
}

// 非法 page_mode 容错为 single（后端同样规范化，双保险）
function normalizePageMode(raw: string | null): PageMode {
  return raw === 'spread' ? 'spread' : 'single'
}

// 排序/文件夹设置状态：本地即时生效，save() 落库（API.md §3.6）
// 双页字段为 spread store 的镜像：由 spread store 写入后随 save() 统一落库
export const useSettingsStore = defineStore('settings', {
  state: () => ({
    path: '' as string,
    sortMode: 'filename' as SortMode,
    sortDirection: 'asc' as SortDirection,
    // regex 模式专属配置：正则表达式 + 多规则链
    regexPattern: '' as string,
    regexRules: [] as SortRule[],
    // 双页阅读选项（SPREAD_ENGINE.md §3.2 默认值）
    pageMode: 'single' as PageMode,
    readOrder: 'left_to_right' as ReadOrder,
    wideRatio: 1.0 as number,
    singleFirstPage: true as boolean,
    singleLastPage: false as boolean,
    saving: false,
  }),
  actions: {
    // 加载目录已保存设置；失败静默回退默认值，不阻断浏览
    async load(path: string) {
      this.path = path
      try {
        const settings = await getFolderSettings(path)
        this.sortMode = settings.sort_mode ?? 'filename'
        this.sortDirection = settings.sort_direction ?? 'asc'
        this.regexPattern = settings.regex_pattern ?? ''
        this.regexRules = parseRegexConfig(settings.regex_config)
        this.pageMode = normalizePageMode(settings.page_mode)
        this.readOrder = settings.read_order === 'right_to_left' ? 'right_to_left' : 'left_to_right'
        this.wideRatio = settings.wide_ratio != null && settings.wide_ratio > 0 ? settings.wide_ratio : 1.0
        this.singleFirstPage = settings.single_first_page ?? true
        this.singleLastPage = settings.single_last_page ?? false
      } catch {
        this.sortMode = 'filename'
        this.sortDirection = 'asc'
        this.regexPattern = ''
        this.regexRules = []
        this.pageMode = 'single'
        this.readOrder = 'left_to_right'
        this.wideRatio = 1.0
        this.singleFirstPage = true
        this.singleLastPage = false
      }
    },
    // 仅更新本地排序（不落库，即时生效由调用方触发重新请求）
    setSort(mode: SortMode, direction: SortDirection) {
      this.sortMode = mode
      this.sortDirection = direction
    },
    // 更新 regex 配置（不落库，由调用方触发重新请求/保存）
    setRegex(pattern: string, rules: SortRule[]) {
      this.regexPattern = pattern
      this.regexRules = rules
    },
    // 镜像 spread store 的选项（不落库，由 spread store 持久化时随 save() 写入）
    applySpread(options: SpreadOptions) {
      this.pageMode = options.pageMode
      this.readOrder = options.readOrder
      this.wideRatio = options.wideRatio
      this.singleFirstPage = options.singleFirstPage
      this.singleLastPage = options.singleLastPage
    },
    // 以当前状态保存设置；regex 字段仅 sort_mode=regex 时携带（API.md §3.6）
    async save() {
      this.saving = true
      try {
        const isRegex = this.sortMode === 'regex'
        return await saveFolderSettings({
          path: this.path,
          sort_mode: this.sortMode,
          sort_direction: this.sortDirection,
          regex_pattern: isRegex ? this.regexPattern : null,
          regex_config:
            isRegex && this.regexRules.length
              ? JSON.stringify({ rules: this.regexRules })
              : null,
          page_mode: this.pageMode,
          read_order: this.readOrder,
          wide_ratio: this.wideRatio,
          single_first_page: this.singleFirstPage,
          single_last_page: this.singleLastPage,
        })
      } finally {
        this.saving = false
      }
    },
  },
})
