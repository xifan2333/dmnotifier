package models

import "strings"

// Platform 直播平台类型
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

// PlatformAliases 别名 → 规范 id
var PlatformAliases = map[string]Platform{
	"bilibili":    PlatformBilibili,
	"bili":        PlatformBilibili,
	"douyin":      PlatformDouyin,
	"dy":          PlatformDouyin,
	"xiaohongshu": PlatformXiaoHongShu,
	"xhs":         PlatformXiaoHongShu,
	"red":         PlatformXiaoHongShu,
	"kuaishou":    PlatformKuaishou,
	"ks":          PlatformKuaishou,
	"douyu":       PlatformDouyu,
	"huya":        PlatformHuya,
}

// String 返回平台名称
func (p Platform) String() string {
	return string(p)
}

// NormalizePlatform 规范化平台 id（支持别名）
func NormalizePlatform(s string) (Platform, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if p, ok := PlatformAliases[s]; ok {
		return p, true
	}
	return "", false
}

// IsValid 是否已知平台
func (p Platform) IsValid() bool {
	_, ok := PlatformAliases[string(p)]
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
