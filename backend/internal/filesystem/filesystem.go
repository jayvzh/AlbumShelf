// Package filesystem 提供受限在 IMAGE_ROOT 内的本地文件访问：
// 路径安全（path.go）、目录扫描（scanner.go）、文件元信息（metadata.go）。
package filesystem

import (
	"os"
	"path/filepath"

	"albumshelf/backend/internal/model"
)

// Filesystem 文件访问统一接口。Sprint 1 定义目录列举，
// Sprint 2 追加图片读取与元信息；未来可替换为 NAS / WebDAV / S3 实现。
type Filesystem interface {
	// ListDirectory 返回目录内单层的图片与子目录；path 为相对 IMAGE_ROOT 的绝对风格路径。
	ListDirectory(path string) ([]model.Image, []model.Folder, error)
	// OpenImage 校验 path 指向的图片文件并返回元信息与已打开的文件句柄；调用方负责 Close。
	OpenImage(path string) (model.Image, *os.File, error)
	// GetMetadata 返回 path 指向图片的元信息，不打开文件。
	GetMetadata(path string) (model.Image, error)
}

// LocalFilesystem 基于本地磁盘的 Filesystem 实现。
type LocalFilesystem struct {
	// root 已做 Abs + EvalSymlinks 解析并缓存的根目录。
	root string
}

// 编译期确认 LocalFilesystem 实现 Filesystem。
var _ Filesystem = (*LocalFilesystem)(nil)

// NewLocalFilesystem 构造 LocalFilesystem，root 解析一次并缓存。
func NewLocalFilesystem(root string) *LocalFilesystem {
	return &LocalFilesystem{root: canonicalRoot(root)}
}

// Root 返回解析后的根目录绝对路径。
func (fs *LocalFilesystem) Root() string {
	return fs.root
}

// ListDirectory 列出目录单层内容：Resolve 校验 → 确认是目录 → 扫描。
// 路径非法返回 ErrInvalidPath；不存在或不是目录返回 ErrNotFound。
func (fs *LocalFilesystem) ListDirectory(path string) ([]model.Image, []model.Folder, error) {
	abs, _, err := Resolve(fs.root, path)
	if err != nil {
		return nil, nil, err
	}

	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return nil, nil, ErrNotFound
	}

	folders, images, err := scanDir(abs, fs.root)
	if err != nil {
		return nil, nil, err
	}
	// 接口契约顺序为 (images, folders)，scanDir 返回 (folders, images)
	return images, folders, nil
}

// OpenImage 打开图片文件：Resolve 校验 → 存在性/文件类型/扩展名校验 → os.Open。
// 返回元信息与文件句柄；路径非法返回 ErrInvalidPath，不存在或是目录返回 ErrNotFound，
// 扩展名非图片格式返回 ErrUnsupported。
func (fs *LocalFilesystem) OpenImage(path string) (model.Image, *os.File, error) {
	abs, err := fs.resolveAndValidateImage(path)
	if err != nil {
		return model.Image{}, nil, err
	}

	file, err := os.Open(abs)
	if err != nil {
		return model.Image{}, nil, ErrNotFound
	}

	img, err := readImageMetadata(abs, fs.root)
	if err != nil {
		file.Close()
		return model.Image{}, nil, err
	}
	return img, file, nil
}

// GetMetadata 返回图片元信息，校验流程同 OpenImage 但不打开文件。
func (fs *LocalFilesystem) GetMetadata(path string) (model.Image, error) {
	abs, err := fs.resolveAndValidateImage(path)
	if err != nil {
		return model.Image{}, err
	}

	return readImageMetadata(abs, fs.root)
}

// resolveAndValidateImage 图片访问共用校验：Resolve 路径安全检查 →
// Stat 确认存在且为普通文件 → 扩展名须为支持的图片格式。
func (fs *LocalFilesystem) resolveAndValidateImage(path string) (string, error) {
	abs, _, err := Resolve(fs.root, path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		return "", ErrNotFound
	}
	if !IsImageExt(filepath.Ext(info.Name())) {
		return "", ErrUnsupported
	}
	return abs, nil
}
