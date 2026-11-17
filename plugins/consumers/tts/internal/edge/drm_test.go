package edge

import (
	"testing"
	"time"
)

// Sec-MS-GEC v2 令牌的固定测试向量。这段算法是微软私有逆向产物（变体
// SHA-256 + CS64，见 drm.go 头注），不允许改动任何常量或轮函数；向量是
// 改动后唯一能立刻发现偏差的护栏。
//
// 期望值来源（三方交叉验证）：
//  1. rany2/edge-tts PR #492 捕获自 Edge 浏览器真实流量的令牌
//     （generate_sec_ms_gec_v2(1791888000)），以及本仓库移植与 Python
//     drm.py 参考逐字节比对的多组时间戳；
//  2. rany2/edge-tts 7.3.1 官方值（经 lew2sss/edge-tts-go-gec-v2 的
//     drm_v2_test.go 引用，标注 "期望值来自 rany2/edge-tts 7.3.1 的
//     generate_sec_ms_gec_v2(unix)，不是自算一遍"）。
func TestSecMsGecV2Vectors(t *testing.T) {
	vectors := map[int64]string{
		// PR #492：2026-10-13T10:40:00Z 5 分钟窗口，Edge 浏览器实测令牌。
		1791888000: "492D7694A183A45E-56E821F95D4579C5B7C7E33DC7303FF379EC36D510213555BBBB9EB17EF15876",
		// 与 Python drm.py 参考互验。
		1766460461: "16CBB3E473DFFBBB-F466D1D29FFA87C8251CC1A0FC0986EC0E66B24F7D6CD9C9BDCA90349BB54527",
		1793000000: "781C1B76C9F739B3-1A772624F17530DF13FBF300637917F70D55C163861ADEC2DA70A248A0DE438A",
		1800000000: "99E54519B490BE5C-E162DB0977FBDCB9B86DD8148BF1BCDC3C0303176CE39852A42B415935C20654",
		// edge-tts 7.3.1 官方期望值。
		1700000000: "2CD23A2CA0B2595A-F3C846C370E05F9CC83ACC5F995C5F5D87A4D0F14CE8936A0F902E64099E38A2",
		1700000100: "0919602B5750900A-EC9DD394F23CB5635CF62A0E8E24136DCD32B26D10C543B10082E482D7516397",
		0:          "52BC7A17B50DFEC9-28C4394775C8751CD2F6859C030C06DB1489ACAB3B3A7651831FC26895177ED6",
	}
	for ts, want := range vectors {
		got, err := generateSecMsGecV2(time.Unix(ts, 0).UTC())
		if err != nil {
			t.Fatalf("ts=%d: %v", ts, err)
		}
		if got != want {
			t.Errorf("ts=%d mismatch:\n got %s\nwant %s", ts, got, want)
		}
	}
}

// 令牌格式必须始终是 <16 hex>-<64 hex>。
func TestSecMsGecV2Shape(t *testing.T) {
	got, err := generateSecMsGecV2(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 16+1+64 {
		t.Fatalf("token %q has length %d, want 81", got, len(got))
	}
}
