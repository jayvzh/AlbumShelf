import { request } from './api'
import type { ImageFile } from '../types/file'
import type { ImageInfo } from '../types/image'

// 版本参数：{mtime}{size}，服务端用于缓存失效；mtime 解析失败时兜底只用 size
function versionOf(image: ImageFile): string {
  const mtime = Date.parse(image.modified_at)
  return `${Number.isNaN(mtime) ? '' : mtime}${image.size}`
}

// 构造图片 URL（含 variant 与版本参数，组件内一律经此函数取图，不手拼）
export function buildImageUrl(
  image: ImageFile,
  variant: 'original' | 'preview' = 'preview',
): string {
  return `/api/v1/image?path=${encodeURIComponent(image.path)}&variant=${variant}&v=${versionOf(image)}`
}

// 缩略图宽度桶按 devicePixelRatio 自适应（<1.5→200、<2.5→300、否则 500），避免 dpr=1 桌面拉高桶浪费带宽；
// 桶值仅 200/300/500（服务端合法桶，见后端 model/thumbnail.go）；SSR/测试环境无 window 时回退 300
function defaultThumbBucket(): number {
  if (typeof window === 'undefined') return 300
  const dpr = window.devicePixelRatio
  if (dpr < 1.5) return 200
  if (dpr < 2.5) return 300
  return 500
}

// 构造缩略图 URL（按宽度缩放，Filmstrip 懒加载用；复用版本参数命中浏览器缓存）
// width 缺省时按 dpr 选桶，显式传参行为不变
export function buildThumbnailUrl(image: ImageFile, width: number = defaultThumbBucket()): string {
  return `/api/v1/thumbnail?path=${encodeURIComponent(image.path)}&width=${width}&v=${versionOf(image)}`
}

// onError 一次性 cache-bust：追加时间戳参数，强制绕过可能已损坏的浏览器缓存条目重新请求
export function withCacheBust(url: string): string {
  return `${url}&cb=${Date.now()}`
}

// MIME → 扩展名：preview 服务端统一 JPEG 输出（生成失败回退原图时 MIME 为真实格式），用于下载文件名修正
const MIME_EXTENSIONS: Record<string, string> = {
  'image/avif': '.avif',
  'image/bmp': '.bmp',
  'image/gif': '.gif',
  'image/jpeg': '.jpg',
  'image/png': '.png',
  'image/webp': '.webp',
}

// 下载文件名：原图保留原文件名；预览产物为 JPEG，按响应 MIME 替换扩展名（拿不到则兜底 .jpg）
function downloadFileName(
  image: ImageFile,
  variant: 'original' | 'preview',
  mime: string,
): string {
  if (variant === 'original') return image.name
  const dot = image.name.lastIndexOf('.')
  const base = dot > 0 ? image.name.slice(0, dot) : image.name
  return `${base}${MIME_EXTENSIONS[mime] ?? '.jpg'}`
}

// 下载图片：请求 URL 与阅读器内 <img> 完全一致，immutable 缓存下直接命中浏览器缓存（即"从缓存下载"），
// 未缓存时同源请求兜底；经 blob + a[download] 触发浏览器下载并保留正确文件名
export async function downloadImage(
  image: ImageFile,
  variant: 'original' | 'preview' = 'preview',
): Promise<void> {
  const response = await fetch(buildImageUrl(image, variant))
  if (!response.ok) throw new Error(`下载失败：HTTP ${response.status}`)
  const blob = await response.blob()
  const objectUrl = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = objectUrl
  anchor.download = downloadFileName(image, variant, blob.type)
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  // 延迟释放：部分浏览器需在下载启动后仍可读取 blob URL
  setTimeout(() => URL.revokeObjectURL(objectUrl), 10_000)
}

// 获取图片元信息
export function getImageInfo(path: string): Promise<ImageInfo> {
  return request<ImageInfo>(`/image/info?path=${encodeURIComponent(path)}`)
}
