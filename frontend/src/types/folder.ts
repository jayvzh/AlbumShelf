import type { ImageFile } from './file'

export interface FolderItem {
  name: string
  path: string
}

export interface FolderResponse {
  path: string
  folders: FolderItem[]
  images: ImageFile[]
}
