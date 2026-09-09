// favorites store 单元测试：拉取填充、乐观切换与回滚、门控 getter、清空
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { ApiError } from '../services/api'

const mocks = vi.hoisted(() => ({
  getFavorites: vi.fn(),
  toggleFavorite: vi.fn(),
}))

vi.mock('../services/favorite.service', () => mocks)

import { useAuthStore } from './auth'
import { useFavoritesStore } from './favorites'
import type { ImageFile } from '../types/file'

function img(path: string): ImageFile {
  return {
    name: path.split('/').pop() ?? path,
    path,
    extension: 'jpg',
    size: 1024,
    width: 400,
    height: 600,
    modified_at: '2026-01-01T00:00:00Z',
  }
}

describe('favorites store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetchAll 填充 images 与 paths', async () => {
    const store = useFavoritesStore()
    mocks.getFavorites.mockResolvedValue({ images: [img('/a/001.jpg'), img('/b/002.jpg')] })

    await store.fetchAll()

    expect(store.loaded).toBe(true)
    expect(store.images).toHaveLength(2)
    expect(store.count).toBe(2)
    expect(store.has('/a/001.jpg')).toBe(true)
    expect(store.has('/b/002.jpg')).toBe(true)
    expect(store.has('/c/003.jpg')).toBe(false)
  })

  it('fetchAll 失败记录 error 且不置 loaded', async () => {
    const store = useFavoritesStore()
    mocks.getFavorites.mockRejectedValue(new ApiError('NETWORK_ERROR', '网络请求失败'))

    await store.fetchAll()

    expect(store.loaded).toBe(false)
    expect(store.error).toContain('NETWORK_ERROR')
  })

  it('toggle 新增：乐观加入 paths，新收藏插入 images 首位', async () => {
    const store = useFavoritesStore()
    store.images = [img('/a/001.jpg')]
    store.paths = new Set(['/a/001.jpg'])
    mocks.toggleFavorite.mockResolvedValue({ path: '/b/002.jpg', favorited: true })

    const favorited = await store.toggle('/b/002.jpg', img('/b/002.jpg'))

    expect(favorited).toBe(true)
    expect(mocks.toggleFavorite).toHaveBeenCalledWith('/b/002.jpg')
    expect(store.has('/b/002.jpg')).toBe(true)
    expect(store.images[0].path).toBe('/b/002.jpg') // 新收藏在前
  })

  it('toggle 取消：从 paths 与 images 移除', async () => {
    const store = useFavoritesStore()
    store.images = [img('/a/001.jpg')]
    store.paths = new Set(['/a/001.jpg'])
    mocks.toggleFavorite.mockResolvedValue({ path: '/a/001.jpg', favorited: false })

    const favorited = await store.toggle('/a/001.jpg')

    expect(favorited).toBe(false)
    expect(store.has('/a/001.jpg')).toBe(false)
    expect(store.images).toHaveLength(0)
  })

  it('toggle 失败回滚到原状态并抛出', async () => {
    const store = useFavoritesStore()
    mocks.toggleFavorite.mockRejectedValue(new ApiError('FILE_NOT_FOUND', 'File not found'))

    await expect(store.toggle('/a/001.jpg', img('/a/001.jpg'))).rejects.toBeInstanceOf(ApiError)

    expect(store.has('/a/001.jpg')).toBe(false)
    expect(store.images).toHaveLength(0)
  })

  it('available：auth 关闭恒可用；开启时仅已登录可用', () => {
    const auth = useAuthStore()
    const store = useFavoritesStore()

    auth.enabled = false
    expect(store.available).toBe(true)

    auth.enabled = true
    auth.authenticated = false
    expect(store.available).toBe(false)

    auth.authenticated = true
    expect(store.available).toBe(true)
  })

  it('clear 重置全部状态', async () => {
    const store = useFavoritesStore()
    store.images = [img('/a/001.jpg')]
    store.paths = new Set(['/a/001.jpg'])
    store.loaded = true

    store.clear()

    expect(store.images).toHaveLength(0)
    expect(store.count).toBe(0)
    expect(store.loaded).toBe(false)
  })
})
