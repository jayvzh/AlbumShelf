<script setup lang="ts">
// 初始化引导页（SPRINT8_TASK.md §5.5）：单页诊断引导，非配置向导——
// 仅展示状态清单与修复指引，不写任何配置（env 是唯一配置源）
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { fetchSetupStatus } from '../services/setup.service'
import { markSetupInitialized } from '../router'
import type { SetupStatus } from '../types/setup'
import { APP_NAME } from '../constants/app'

const router = useRouter()

const status = ref<SetupStatus | null>(null)
const error = ref('')
const rechecking = ref(false)

onMounted(refresh)

// 重新检测：initialized 时更新守卫 memo 并进入应用
async function refresh() {
  if (rechecking.value) return
  rechecking.value = true
  error.value = ''
  try {
    status.value = await fetchSetupStatus()
    if (status.value.initialized) {
      markSetupInitialized()
      router.push('/')
    }
  } catch {
    error.value = '检测请求失败，请确认服务正在运行后重试'
  } finally {
    rechecking.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-base px-4 py-10">
    <div class="w-[42rem] max-w-full rounded-lg border border-line-strong bg-panel p-8 shadow-xl">
      <!-- 标题区 -->
      <h1 class="text-lg font-semibold text-ink">欢迎使用 {{ APP_NAME }}</h1>
      <p class="mt-1 text-xs text-muted">服务已启动，但图片目录尚未就绪。请按以下检查项排查部署配置。</p>

      <!-- 状态清单 -->
      <div v-if="status" class="mt-6 space-y-2.5">
        <div class="flex items-center gap-2.5 text-sm">
          <span :class="status.image_root_configured ? 'text-green-600' : 'text-red-500'" class="w-4 shrink-0 text-center font-semibold">
            {{ status.image_root_configured ? '✓' : '✗' }}
          </span>
          <span class="text-body">环境变量 IMAGE_ROOT 已配置</span>
        </div>
        <div class="flex items-center gap-2.5 text-sm">
          <span :class="status.image_root_exists ? 'text-green-600' : 'text-red-500'" class="w-4 shrink-0 text-center font-semibold">
            {{ status.image_root_exists ? '✓' : '✗' }}
          </span>
          <span class="text-body">图片目录已挂载且存在</span>
        </div>
        <div class="flex items-center gap-2.5 text-sm">
          <span :class="status.image_root_readable ? 'text-green-600' : 'text-red-500'" class="w-4 shrink-0 text-center font-semibold">
            {{ status.image_root_readable ? '✓' : '✗' }}
          </span>
          <span class="text-body">图片目录可读</span>
        </div>
        <!-- 信息行（非阻塞，不参与 initialized） -->
        <div class="flex items-center gap-2.5 text-sm">
          <span class="w-4 shrink-0 text-center text-faint">·</span>
          <span class="text-muted">
            登录系统：{{ status.auth_enabled ? '已启用（用户名 AUTH_USERNAME）' : '未启用（游客模式）' }}
          </span>
        </div>
      </div>

      <p v-else-if="error" class="mt-6 text-xs text-accent-text">{{ error }}</p>
      <p v-else class="mt-6 text-xs text-muted">正在检测初始化状态…</p>

      <!-- 修复指引：仅渲染未通过项 -->
      <div v-if="status" class="mt-6 space-y-5">
        <div v-if="!status.image_root_configured">
          <h2 class="text-sm font-medium text-ink">修复 IMAGE_ROOT 未配置</h2>
          <p class="mt-1 text-xs text-muted">正常部署不会出现此项；请确认 docker-compose.yml 的 environment 段：</p>
          <pre class="mt-2 overflow-x-auto rounded border border-line bg-elevated p-3 text-xs text-body">environment:
  - IMAGE_ROOT=/images</pre>
        </div>

        <div v-if="!status.image_root_exists">
          <h2 class="text-sm font-medium text-ink">修复图片目录不存在</h2>
          <p class="mt-1 text-xs text-muted">确认 docker-compose.yml 的 volumes 左侧指向宿主机上真实存在的照片目录：</p>
          <pre class="mt-2 overflow-x-auto rounded border border-line bg-elevated p-3 text-xs text-body">volumes:
  - /path/to/your/photos:/images:ro   # 左侧改为宿主机照片目录</pre>
          <p class="mt-2 text-xs text-muted">修改后需重建容器生效：<code class="rounded bg-elevated px-1.5 py-0.5">docker compose up -d</code></p>
        </div>

        <div v-if="!status.image_root_readable">
          <h2 class="text-sm font-medium text-ink">修复图片目录不可读</h2>
          <p class="mt-1 text-xs text-muted">为目录授予读取与遍历权限（宿主机执行）：</p>
          <pre class="mt-2 overflow-x-auto rounded border border-line bg-elevated p-3 text-xs text-body">chmod -R a+rX /path/to/your/photos
# 或调整属主（容器以 root 运行，任意属主可读即可）：
chown -R 0:0 /path/to/your/photos</pre>
          <p class="mt-2 text-xs text-muted">
            群晖 / 威联通：在共享文件夹的权限设置中，为对应账号勾选「可读 / 可遍历」，
            并确认 NAS 挂载（NFS/CIFS）本身允许读取。
          </p>
        </div>
      </div>

      <!-- 操作 -->
      <div class="mt-7 flex items-center gap-3">
        <button
          type="button"
          :disabled="rechecking"
          class="rounded border border-line-strong bg-elevated px-4 py-2 text-sm text-ink transition-colors hover:border-line-hover hover:bg-elevated-hover disabled:cursor-not-allowed disabled:opacity-60"
          @click="refresh"
        >
          {{ rechecking ? '检测中…' : '重新检测' }}
        </button>
        <span v-if="error" class="text-xs text-accent-text">{{ error }}</span>
      </div>

      <!-- 底部排查提示 -->
      <div class="mt-6 border-t border-line pt-4 text-xs text-muted">
        <p>仍无法解决？查看容器日志定位问题：<code class="rounded bg-elevated px-1.5 py-0.5">docker compose logs imageshelf</code></p>
        <p class="mt-1">修改 <code class="rounded bg-elevated px-1.5 py-0.5">.env</code> 或 docker-compose.yml 后，需 <code class="rounded bg-elevated px-1.5 py-0.5">docker compose up -d</code> 重建容器生效。</p>
      </div>
    </div>
  </div>
</template>
