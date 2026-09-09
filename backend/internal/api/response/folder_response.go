package response

import (
	"time"

	"imageshelf/backend/internal/model"
)

// FolderData GET /api/v1/folders 成功响应（API.md §3.2）。
type FolderData struct {
	Path    string       `json:"path"`
	Folders []FolderItem `json:"folders"`
	Images  []ImageFile  `json:"images"`
}

// FolderItem 子目录条目。
type FolderItem struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ImageFile 图片文件条目；width/height 解析失败为 null。
type ImageFile struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Extension  string `json:"extension"`
	Size       int64  `json:"size"`
	Width      *int   `json:"width"`
	Height     *int   `json:"height"`
	ModifiedAt string `json:"modified_at"` // UTC RFC3339
}

// NewFolderData 把领域模型转换为响应 DTO；空列表序列化为 [] 而非 null。
func NewFolderData(path string, folders []model.Folder, images []model.Image) FolderData {
	dtoFolders := make([]FolderItem, 0, len(folders))
	for _, f := range folders {
		dtoFolders = append(dtoFolders, FolderItem{Name: f.Name, Path: f.Path})
	}
	return FolderData{Path: path, Folders: dtoFolders, Images: NewImageFiles(images)}
}

// NewImageFiles 图片条目 DTO 批量转换；空列表序列化为 [] 而非 null。
func NewImageFiles(images []model.Image) []ImageFile {
	dtoImages := make([]ImageFile, 0, len(images))
	for _, img := range images {
		dtoImages = append(dtoImages, ImageFile{
			Name:       img.Name,
			Path:       img.Path,
			Extension:  img.Extension,
			Size:       img.Size,
			Width:      img.Width,
			Height:     img.Height,
			ModifiedAt: img.ModifiedAt.UTC().Format(time.RFC3339),
		})
	}
	return dtoImages
}
