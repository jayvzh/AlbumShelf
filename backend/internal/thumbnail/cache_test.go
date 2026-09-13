package thumbnail

import (
	"strings"
	"testing"

	"albumshelf/backend/internal/model"
)

// 同输入两次调用必须返回同一路径（键稳定）。
func TestCachePathDeterministic(t *testing.T) {
	args := cacheArgs{dataDir: t.TempDir(), source: "/Comics/001.jpg", variant: model.VariantThumb, bucket: 300, mtime: 123, size: 456}
	p1, err := CachePath(args.dataDir, args.source, args.variant, args.bucket, args.mtime, args.size)
	if err != nil {
		t.Fatalf("CachePath 出错: %v", err)
	}
	p2, err := CachePath(args.dataDir, args.source, args.variant, args.bucket, args.mtime, args.size)
	if err != nil {
		t.Fatalf("CachePath 出错: %v", err)
	}
	if p1 != p2 {
		t.Fatalf("同输入键不稳定: %s != %s", p1, p2)
	}
}

type cacheArgs struct {
	dataDir, source, variant string
	bucket                   int
	mtime, size              int64
}

// variant / bucket / mtime / size 任一变化都必须产生不同路径（失效语义）。
func TestCachePathSensitiveToInputs(t *testing.T) {
	base := cacheArgs{dataDir: t.TempDir(), source: "/Comics/001.jpg", variant: model.VariantThumb, bucket: 300, mtime: 123, size: 456}
	basePath, err := CachePath(base.dataDir, base.source, base.variant, base.bucket, base.mtime, base.size)
	if err != nil {
		t.Fatalf("CachePath 出错: %v", err)
	}

	cases := map[string]cacheArgs{
		"variant变化": {base.dataDir, base.source, model.VariantPreview, base.bucket, base.mtime, base.size},
		"bucket变化":  {base.dataDir, base.source, base.variant, 500, base.mtime, base.size},
		"mtime变化":   {base.dataDir, base.source, base.variant, base.bucket, 124, base.size},
		"size变化":    {base.dataDir, base.source, base.variant, base.bucket, base.mtime, 789},
	}
	for name, args := range cases {
		p, err := CachePath(args.dataDir, args.source, args.variant, args.bucket, args.mtime, args.size)
		if err != nil {
			t.Fatalf("%s: CachePath 出错: %v", name, err)
		}
		if p == basePath {
			t.Fatalf("%s: 路径未变化", name)
		}
	}
}

// thumb 与 preview 必须落到各自子目录 dataDir/cache/{thumb,preview}。
func TestCachePathVariantSubDir(t *testing.T) {
	dataDir := t.TempDir()
	thumbPath, err := CachePath(dataDir, "/a.jpg", model.VariantThumb, 300, 1, 2)
	if err != nil {
		t.Fatalf("CachePath 出错: %v", err)
	}
	previewPath, err := CachePath(dataDir, "/a.jpg", model.VariantPreview, 0, 1, 2)
	if err != nil {
		t.Fatalf("CachePath 出错: %v", err)
	}
	if want := dataDir + "/cache/thumb/"; !strings.HasPrefix(thumbPath, want) || !strings.HasSuffix(thumbPath, ".jpg") {
		t.Fatalf("thumb 路径不在 cache/thumb 下: %s", thumbPath)
	}
	if want := dataDir + "/cache/preview/"; !strings.HasPrefix(previewPath, want) || !strings.HasSuffix(previewPath, ".jpg") {
		t.Fatalf("preview 路径不在 cache/preview 下: %s", previewPath)
	}
}

// 未知变体必须报错。
func TestCachePathInvalidVariant(t *testing.T) {
	if _, err := CachePath(t.TempDir(), "/a.jpg", "evil", 300, 1, 2); err == nil {
		t.Fatal("未知变体应返回错误")
	}
}
