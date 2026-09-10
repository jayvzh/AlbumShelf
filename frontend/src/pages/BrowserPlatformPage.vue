<script setup lang="ts">
// 平台分流包装页：同一 / 路由按视口断点选择桌面或移动组件树。
// 双组件树架构——桌面（≥768px）渲染 BrowserPage，移动渲染 MobileBrowserPage，
// 逻辑层（stores/services/composables/utils）100% 共享，桌面文件零改动。
import { watchEffect } from 'vue'
import { useIsMobile, useMediaQuery } from '../composables/useMediaQuery'
import BrowserPage from './BrowserPage.vue'
import MobileBrowserPage from './mobile/MobileBrowserPage.vue'

const isMobile = useIsMobile()
// 触屏设备（primary pointer 为手指）：为真且渲染桌面树 = 手机/平板上"请求桌面版"
const touchPointer = useMediaQuery('(pointer: coarse)')

// 手机浏览器"请求桌面版"会把 layout viewport 拉到 980px，但下方默认 meta 的
// initial-scale=1 + maximum-scale=1 + user-scalable=no 锁死缩放（为移动版阅读器
// 双指手势设计），导致桌面布局 1:1 局部显示、页面虚高、底部 Filmstrip 沉出视口。
// 按树分流动态切换 meta：触屏渲染桌面树时放开缩放锁，浏览器自动整页缩览；
// 移动树维持锁定，保证阅读器手势不被浏览器页面缩放干扰。
const MOBILE_META =
  'width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no, viewport-fit=cover'
const DESKTOP_META = 'width=980, viewport-fit=cover'

watchEffect(() => {
  const meta = document.querySelector<HTMLMetaElement>('meta[name="viewport"]')
  if (!meta) return
  meta.setAttribute('content', !isMobile.value && touchPointer.value ? DESKTOP_META : MOBILE_META)
})
</script>

<template>
  <MobileBrowserPage v-if="isMobile" />
  <BrowserPage v-else />
</template>
