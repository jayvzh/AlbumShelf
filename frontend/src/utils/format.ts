// 文件大小格式化：B/KB/MB/GB，1 位小数（如 1.2 MB）；
// compact 用于按钮内体积提示：无小数无空格单字母单位（如 2M / 560K）
export function formatFileSize(bytes: number, compact = false): string {
  const units = compact ? ['B', 'K', 'M', 'G'] : ['B', 'KB', 'MB', 'GB']
  let size = bytes
  let i = 0
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  if (compact) return `${Math.round(size)}${units[i]}`
  return `${i === 0 ? size : size.toFixed(1)} ${units[i]}`
}
