// 文件大小格式化：B/KB/MB/GB，1 位小数（如 1.2 MB）
export function formatFileSize(bytes: number): string {
  const units = ['B', 'KB', 'MB', 'GB']
  let size = bytes
  let i = 0
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${i === 0 ? size : size.toFixed(1)} ${units[i]}`
}
