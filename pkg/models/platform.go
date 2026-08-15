package models

import "strings"

// Platform 直播平台类型（与 UniBarrage 规范 id 一致，无别名）
type Platform string

const (
	PlatformDouyin      Platform = "douyin"
	PlatformBilibili    Platform = "bilibili"
	PlatformKuaishou    Platform = "kuaishou"
	PlatformDouyu       Platform = "douyu"
	PlatformHuya        Platform = "huya"
	PlatformXiaoHongShu Platform = "xiaohongshu"
)

// AllPlatforms 有序列表（CLI / TUI 共用）
var AllPlatforms = []Platform{
	PlatformBilibili,
	PlatformDouyin,
	PlatformXiaoHongShu,
	PlatformKuaishou,
	PlatformDouyu,
	PlatformHuya,
}

// String 返回平台名称
func (p Platform) String() string {
	return string(p)
}

// ParsePlatform 解析规范平台 id（大小写不敏感，不接受别名）
func ParsePlatform(s string) (Platform, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, p := range AllPlatforms {
		if string(p) == s {
			return p, true
		}
	}
	return "", false
}

// IsValid 是否已知平台
func (p Platform) IsValid() bool {
	_, ok := ParsePlatform(string(p))
	return ok
}

// PlatformStrings 返回字符串切片
func PlatformStrings() []string {
	out := make([]string, len(AllPlatforms))
	for i, p := range AllPlatforms {
		out[i] = string(p)
	}
	return out
}
