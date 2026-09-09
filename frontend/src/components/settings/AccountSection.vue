<script setup lang="ts">
// 账户与系统信息区块（SPRINT7_TASK.md §5.11）：登录态/退出 + 服务器路径只读展示
// （image_root/data_dir 来自 /auth/status 已登录字段；auth disabled 时整块降级提示）
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { fetchAuthStatus } from '../../services/auth.service'

const router = useRouter()
const auth = useAuthStore()

// 系统信息仅已登录时可见（游客请求不返回服务器路径）
const imageRoot = ref('')
const dataDir = ref('')

onMounted(async () => {
  if (!auth.enabled || !auth.authenticated) return
  try {
    const status = await fetchAuthStatus()
    imageRoot.value = status.image_root ?? ''
    dataDir.value = status.data_dir ?? ''
  } catch {
    // 状态获取失败不阻断账户区展示
  }
})

async function logout() {
  try {
    await auth.logout()
  } finally {
    router.push('/')
  }
}
</script>

<template>
  <section class="rounded-lg border border-line bg-panel p-4">
    <h2 class="text-sm font-medium text-ink">账户与系统信息</h2>

    <!-- 登录系统未启用：不显示系统信息（不向游客泄露服务器路径） -->
    <p v-if="!auth.enabled" class="mt-3 text-sm text-muted">登录系统未启用（未设置 AUTH_PASSWORD）</p>

    <template v-else>
      <div class="mt-3 flex items-center gap-3">
        <span class="text-sm text-body">
          {{ auth.authenticated ? `已登录：${auth.username}` : '未登录' }}
        </span>
        <button
          v-if="auth.authenticated"
          type="button"
          class="rounded border border-line-strong bg-elevated px-3 py-1.5 text-sm text-body transition-colors hover:border-line-hover hover:text-ink"
          @click="logout"
        >
          退出登录
        </button>
      </div>

      <dl v-if="auth.authenticated" class="mt-3 space-y-1">
        <div class="flex gap-2">
          <dt class="w-24 shrink-0 text-xs text-muted">图片根目录</dt>
          <dd class="min-w-0 truncate font-mono text-xs text-body" :title="imageRoot">
            {{ imageRoot || '—' }}
          </dd>
        </div>
        <div class="flex gap-2">
          <dt class="w-24 shrink-0 text-xs text-muted">数据目录</dt>
          <dd class="min-w-0 truncate font-mono text-xs text-body" :title="dataDir">
            {{ dataDir || '—' }}
          </dd>
        </div>
      </dl>
    </template>
  </section>
</template>
