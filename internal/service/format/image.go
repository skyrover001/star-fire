package format

import (
	"encoding/json"
	"strings"
)

// 本文件存放图片 data URL 与 Anthropic image source 对象之间的转换辅助函数。
//
// Canonical 侧统一用标准 data URL（data:{media_type};base64,{data}）或 http(s)
// URL 表达图片；Anthropic 侧用 source 对象（{type:base64, media_type, data} 或
// {type:url, url}）。

// anthropicImageSource 是 Anthropic image block 的 source 对象。
type anthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
}

// sourceToImageURL 把 Anthropic image source 转成标准 image URL：
// base64 → data:{media_type};base64,{data}；url → 原样 URL。
func sourceToImageURL(source json.RawMessage) string {
	var src anthropicImageSource
	if err := json.Unmarshal(source, &src); err != nil {
		return ""
	}
	switch src.Type {
	case "base64":
		return "data:" + src.MediaType + ";base64," + src.Data
	case "url":
		return src.URL
	default:
		return ""
	}
}

// imageURLToSource 把标准 image URL 转成 Anthropic image source：
// data: URL → base64 对象；否则 → url 对象。
func imageURLToSource(imageURL string) json.RawMessage {
	if strings.HasPrefix(imageURL, "data:") {
		if mediaType, data, ok := splitDataURL(imageURL); ok {
			b, _ := json.Marshal(anthropicImageSource{Type: "base64", MediaType: mediaType, Data: data})
			return b
		}
	}
	b, _ := json.Marshal(anthropicImageSource{Type: "url", URL: imageURL})
	return b
}

// splitDataURL 解析 data:{media_type};base64,{data}，返回 media_type 与 base64 数据。
func splitDataURL(dataURL string) (mediaType, data string, ok bool) {
	if !strings.HasPrefix(dataURL, "data:") {
		return "", "", false
	}
	rest := dataURL[len("data:"):]
	idx := strings.Index(rest, ";base64,")
	if idx < 0 {
		return "", "", false
	}
	return rest[:idx], rest[idx+len(";base64,"):], true
}
