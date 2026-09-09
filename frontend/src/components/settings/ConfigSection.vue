<script setup lang="ts">
// 配置导入导出区块（SPRINT7_TASK.md §5.11）：导出直接下载；导入选择 .json 文件后
// 经 AppModal 明确提示合并语义，确认后上传。失败按错误码提示（CONFIG_INVALID / INVALID_PATH）
import { ref } from 'vue'
import { exportConfigUrl, importConfig } from '../../services/appsettings.service'
import { ApiError } from '../../services/api'
import AppModal from '../common/AppModal.vue'

const fileInput = ref<HTMLInputElement | null>(null)
// 待导入文件（文件选择后先暂存，确认后上传）
const pendingFile = ref<File | null>(null)
const importing = ref(false)
const notice = ref('')

function exportConfig() {
  // 响应带 attachment 头，浏览器直接下载（window.open 保留下载行为）
  window.open(exportConfigUrl())
}

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = '' // 允许重复选择同一文件
  if (!file) return
  notice.value = ''
  pendingFile.value = file
}

async function doImport() {
  const file = pendingFile.value
  if (!file || importing.value) return
  importing.value = true
  notice.value = ''
  try {
    const json = await file.text()
    const result = await importConfig(json)
    notice.value = `导入成功：目录设置 ${result.folder_settings} 条，私有目录 ${result.protected_folders} 条`
  } catch (e) {
    if (e instanceof ApiError) {
      notice.value =
        e.code === 'CONFIG_INVALID'
          ? '导入失败：配置文件无效（版本不符或格式错误）'
          : `导入失败（${e.code}）：${e.message}`
    } else {
      notice.value = '导入失败，请重试'
    }
  } finally {
    importing.value = false
    pendingFile.value = null
  }
}
</script>

<template>
  <section class="rounded-lg border border-line bg-panel p-4">
    <h2 class="text-sm font-medium text-ink">配置导入导出</h2>
    <p class="mt-1 text-xs text-muted">
      导出包含目录设置与私有目录；导入为按路径合并（不删除已有条目）。主题等本地偏好不参与。
    </p>

    <div class="mt-3 flex items-center gap-3">
      <button
        type="button"
        class="rounded border border-line-strong bg-elevated px-3 py-1.5 text-sm text-body transition-colors hover:border-line-hover hover:text-ink"
        @click="exportConfig"
      >
        导出配置
      </button>
      <button
        type="button"
        :disabled="importing"
        class="rounded border border-line-strong bg-elevated px-3 py-1.5 text-sm text-body transition-colors hover:border-line-hover hover:text-ink disabled:cursor-not-allowed disabled:opacity-60"
        @click="fileInput?.click()"
      >
        {{ importing ? '导入中…' : '导入配置' }}
      </button>
      <!-- 隐藏文件选择：接受导出的 .json 文件 -->
      <input
        ref="fileInput"
        type="file"
        accept=".json,application/json"
        class="hidden"
        @change="onFileChange"
      />
      <span v-if="notice" class="text-xs text-muted">{{ notice }}</span>
    </div>

    <AppModal
      :open="pendingFile !== null"
      title="导入配置"
      :message="`将导入 ${pendingFile?.name ?? ''}：目录设置与私有目录按路径合并到当前配置。是否继续？`"
      confirm-text="导入"
      @confirm="doImport"
      @close="pendingFile = null"
    />
  </section>
</template>
