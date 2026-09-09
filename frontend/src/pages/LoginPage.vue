<script setup lang="ts">
// 登录页（SPRINT7_TASK.md §3A'）：单管理员登录表单；成功后按 redirect 参数跳回（默认 /）
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { ApiError } from '../services/api'
import { APP_NAME } from '../constants/app'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const error = ref('')
const submitting = ref(false)

async function submit() {
  if (submitting.value) return
  if (!username.value.trim() || !password.value) {
    error.value = '请输入用户名与密码'
    return
  }
  submitting.value = true
  error.value = ''
  try {
    await auth.login(username.value.trim(), password.value)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    router.push(redirect)
  } catch (e) {
    if (e instanceof ApiError) {
      error.value = e.code === 'INVALID_CREDENTIALS' ? '用户名或密码错误' : e.message
    } else {
      error.value = '登录失败，请重试'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="flex h-screen items-center justify-center bg-base">
    <form
      class="w-80 max-w-[92vw] rounded-lg border border-line-strong bg-panel p-6 shadow-xl"
      @submit.prevent="submit"
    >
      <h1 class="text-lg font-semibold text-ink">{{ APP_NAME }}</h1>
      <p class="mt-1 text-xs text-muted">请登录管理员账户</p>

      <label class="mt-5 block text-xs font-medium text-muted" for="login-username">用户名</label>
      <input
        id="login-username"
        v-model="username"
        type="text"
        autocomplete="username"
        class="mt-1 w-full rounded border border-line-strong bg-elevated px-3 py-2 text-sm text-ink outline-none transition-colors focus:border-line-hover"
      />

      <label class="mt-3 block text-xs font-medium text-muted" for="login-password">密码</label>
      <input
        id="login-password"
        v-model="password"
        type="password"
        autocomplete="current-password"
        class="mt-1 w-full rounded border border-line-strong bg-elevated px-3 py-2 text-sm text-ink outline-none transition-colors focus:border-line-hover"
      />

      <p v-if="error" class="mt-3 text-xs text-accent-text">{{ error }}</p>

      <button
        type="submit"
        :disabled="submitting"
        class="mt-4 w-full rounded border border-line-strong bg-elevated px-3 py-2 text-sm text-ink transition-colors hover:border-line-hover hover:bg-elevated-hover disabled:cursor-not-allowed disabled:opacity-60"
      >
        {{ submitting ? '登录中…' : '登录' }}
      </button>

      <button
        type="button"
        class="mt-2 w-full rounded px-3 py-1.5 text-xs text-muted transition-colors hover:text-ink"
        @click="router.push('/')"
      >
        返回浏览
      </button>
    </form>
  </div>
</template>
