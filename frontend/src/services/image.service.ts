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

// 构造缩略图 URL（按宽度缩放，Filmstrip 懒加载用；复用版本参数命中浏览器缓存）
export function buildThumbnailUrl(image: ImageFile, width = 300): string {
  return `/api/v1/thumbnail?path=${encodeURIComponent(image.path)}&width=${width}&v=${versionOf(image)}`
}

// 获取图片元信息
export function getImageInfo(path: string): Promise<ImageInfo> {
  return request<ImageInfo>(`/image/info?path=${encodeURIComponent(path)}`)
}
