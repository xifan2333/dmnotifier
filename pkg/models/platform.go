package models

// Platform 直播平台类型
type Platform string

const (
	PlatformDouyin   Platform = "douyin"
	PlatformBilibili Platform = "bilibili"
	PlatformKuaishou Platform = "kuaishou"
	PlatformDouyu    Platform = "douyu"
	PlatformHuya     Platform = "huya"
)

// String 返回平台名称
func (p Platform) String() string {
	return string(p)
}
