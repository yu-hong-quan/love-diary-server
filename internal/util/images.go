// Package util 提供日记图片字段的兼容处理（Mongo 迁移遗留 image / images 双字段）。
package util

// NormalizeDiaryImages 读取数据库后合并字段：若仅有 image 则并入 images，供 API 返回。
func NormalizeDiaryImages(image string, images []string) []string {
	out := make([]string, 0, len(images)+1)
	if len(images) == 0 && image != "" {
		out = append(out, image)
	} else {
		for _, img := range images {
			if img != "" {
				out = append(out, img)
			}
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

// PrepareDiaryImages 写入数据库前规范化：优先使用 images，否则将 image 转为单元素数组。
func PrepareDiaryImages(image string, images []string) []string {
	if len(images) == 0 && image != "" {
		return []string{image}
	}
	if images == nil {
		return []string{}
	}
	return images
}
