package storage

import "strings"

// HasDataURL 列表中是否包含需落盘的 base64 data URL。
func HasDataURL(images []string) bool {
	for _, img := range images {
		if strings.HasPrefix(strings.TrimSpace(img), "data:") {
			return true
		}
	}
	return false
}
