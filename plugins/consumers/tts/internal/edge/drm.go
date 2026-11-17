package edge

// Sec-MS-GEC v2 token generation.
//
// Ported 1:1 from rany2/edge-tts 7.3.0 (src/edge_tts/drm.py, PR #492). Microsoft
// made the v2 token format mandatory for the Read-Aloud endpoint in October
// 2026: handshakes carrying the legacy v1 token (a plain SHA-256 of
// "<ticks><client token>" with Sec-MS-GEC-Version "1-...") are rejected with
// HTTP 403. The v2 token digests the same string with a tweaked SHA-256 (salt
// folded into the initial state) plus a proprietary CS64 checksum, producing
// "<16 hex>-<64 hex>" sent with Sec-MS-GEC-Version "2-...".

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// unix 秒到 Windows 文件时间 (1601-01-01) 的差值。
const winEpoch = 11644473600

var (
	// _SHA256_INITIAL：标准 SHA-256 初始链接值。
	sha256Initial = [8]uint32{0x6A09E667, 0xBB67AE85, 0x3C6EF372, 0xA54FF53A, 0x510E527F, 0x9B05688C, 0x1F83D9AB, 0x5BE0CD19}
	// _SHA256_STATE：v2 变体使用的初始状态。
	sha256State = [8]uint32{0xFD9FBEA2, 0x91243CE7, 0xBE499094, 0xCD2E1E0A, 0x6C006F49, 0x04BD35AB, 0xFEB0A7E4, 0x03EFFBD2}
	// _SHA256_SCHEDULE：消息调度种子字。
	sha256Schedule = [14]uint32{0xEB24BF9E, 0x1558691F, 0xA0C09EE2, 0x0CFA32BA, 0xA622EE8C, 0xD1DD9491, 0xC0DD25A0, 0xF9E70AB6, 0xF0C31573, 0x7058CE4E, 0x7188E606, 0x00E5AB33, 0xD258B060, 0x72766564}
	// _SHA256_ROUND_CONSTANTS：标准 SHA-256 K 表。
	sha256RoundConstants = [64]uint32{
		0x428A2F98, 0x71374491, 0xB5C0FBCF, 0xE9B5DBA5, 0x3956C25B, 0x59F111F1, 0x923F82A4, 0xAB1C5ED5,
		0xD807AA98, 0x12835B01, 0x243185BE, 0x550C7DC3, 0x72BE5D74, 0x80DEB1FE, 0x9BDC06A7, 0xC19BF174,
		0xE49B69C1, 0xEFBE4786, 0x0FC19DC6, 0x240CA1CC, 0x2DE92C6F, 0x4A7484AA, 0x5CB0A9DC, 0x76F988DA,
		0x983E5152, 0xA831C66D, 0xB00327C8, 0xBF597FC7, 0xC6E00BF3, 0xD5A79147, 0x06CA6351, 0x14292967,
		0x27B70A85, 0x2E1B2138, 0x4D2C6DFC, 0x53380D13, 0x650A7354, 0x766A0ABB, 0x81C2C92E, 0x92722C85,
		0xA2BFE8A1, 0xA81A664B, 0xC24B8B70, 0xC76C51A3, 0xD192E819, 0xD6990624, 0xF40E3585, 0x106AA070,
		0x19A4C116, 0x1E376C08, 0x2748774C, 0x34B0BCB5, 0x391C0CB3, 0x4ED8AA4A, 0x5B9CCA4F, 0x682E6FF3,
		0x748F82EE, 0x78A5636F, 0x84C87814, 0x8CC70208, 0x90BEFFFA, 0xA4506CEB, 0xBEF9A3F7, 0xC67178F2,
	}
	// 第 14 轮常量被调整值替换。
	sha256Adjustment = uint32(0xC9DC06A7)
	// 调度第 16 / 29 词的 sigma 异或调整。
	sha256Sigma0Adjustment = uint32(0x059C0B80)
	sha256Sigma1Adjustment = uint32(0x000B92C0)
	// _CS64_COEFFICIENTS：CS64 摘要的系数表。
	cs64Coefficients = [14][2]uint32{
		{0xE82E91B1, 0xB32700D0}, {0x44DE24C6, 0x88DFF51D}, {0x3BEB65BC, 0x29514038}, {0xC89C5933, 0xB328CBDF},
		{0x8D1EC211, 0xD405B4D5}, {0x314EF5D5, 0x26E55F3C}, {0xE92491B1, 0x607801C6}, {0xF0A5B9D5, 0x82B2C1FB},
		{0x06064CF3, 0xFA51A3DA}, {0xDA79FA60, 0x207CC464}, {0xBED20560, 0x3C3C74F3}, {0x89F9C9F8, 0xD2D14B61},
		{0x35FA38C6, 0xBB04A2C1}, {0xCB49E3C6, 0xC5626167},
	}
	cs64Factor     = uint32(0x82893933)
	cs64Adjustment = [2]uint32{0xF2F90B1A, 0x0B1A0000}
)

// generateSecMsGecV2 计算 <16 hex>-<64 hex> 的 v2 令牌。now 为令牌的有效时间源。
func generateSecMsGecV2(now time.Time) (string, error) {
	// Windows 文件时间（100ns 单位），向下取整到 5 分钟窗口。
	ticks := now.Unix() + winEpoch
	ticks -= ticks % 300
	ticks *= 10000000
	return tokenHash([]byte(fmt.Sprintf("%d%s", ticks, clientToken)))
}

// tokenHash 计算 v2 摘要对：CS64（16 hex）+ 变体 SHA-256（64 hex）。
func tokenHash(data []byte) (string, error) {
	digest, err := sha256Digest(data)
	if err != nil {
		return "", err
	}
	swapLast, swapAcc := cs64Digest(data, digest)
	var prefix [8]byte
	binary.LittleEndian.PutUint32(prefix[0:4], swapLast)
	binary.LittleEndian.PutUint32(prefix[4:8], swapAcc)
	return strings.ToUpper(hex.EncodeToString(prefix[:])) + "-" + strings.ToUpper(hex.EncodeToString(digest)), nil
}

// ---- 变体 SHA-256 ----

func rol16(v uint32) uint32 { return v<<16 | v>>16 }

// 标准 SHA-256 小 sigma（Go 无符号回绕等价于 Python 末尾掩码）。
func sigma0(v uint32) uint32 { return v>>7 ^ v<<25 ^ v>>18 ^ v<<14 ^ v>>3 }
func sigma1(v uint32) uint32 { return v>>17 ^ v<<15 ^ v>>19 ^ v<<13 ^ v>>10 }
func rotr(v uint32, n uint) uint32 {
	return v>>n | v<<(32-n)
}

// sha256Rounds 从 firstRound 运行压缩轮次；firstRound==14 时第 14 轮常量被替换。
func sha256Rounds(state [8]uint32, words []uint32, firstRound int) [8]uint32 {
	a, b, c, d, e, f, g, h := state[0], state[1], state[2], state[3], state[4], state[5], state[6], state[7]
	k := sha256RoundConstants
	if firstRound == 14 {
		k[14] = sha256Adjustment
	}
	for i := firstRound; i < 64; i++ {
		s1 := rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25)
		ch := (e & f) ^ (^e & g)
		t1 := h + s1 + ch + k[i] + words[i]
		s0 := rotr(a, 2) ^ rotr(a, 13) ^ rotr(a, 22)
		maj := (a & b) ^ (a & c) ^ (b & c)
		t2 := s0 + maj
		h, g, f, e, d, c, b, a = g, f, e, d+t1, c, b, a, t1+t2
	}
	return [8]uint32{a, b, c, d, e, f, g, h}
}

func expandSchedule(words []uint32, first int) []uint32 {
	for i := first; i < 64; i++ {
		x, y := words[i-2], words[i-15]
		s1 := x>>17 ^ x<<15 ^ x>>19 ^ x<<13 ^ x>>10
		s0 := y>>7 ^ y<<25 ^ y>>18 ^ y<<14 ^ y>>3
		words = append(words, s1+words[i-7]+s0+words[i-16])
	}
	return words
}

// sha256Init 把前 7 字节盐折叠进初始状态，词表按调整常量构造。
func sha256Init(salt []byte) [8]uint32 {
	words := make([]uint32, 14, 64)
	words = append(words, uint32(salt[0])<<16|uint32(salt[1])<<8|uint32(salt[2]))
	words = append(words, binary.BigEndian.Uint32(salt[3:7]))
	for index := 16; index < 30; index++ {
		value := sha256Schedule[index-16]
		if index-2 >= 14 {
			s := sigma1(words[index-2])
			if index == 16 {
				s ^= sha256Sigma1Adjustment
			}
			value += s
		}
		if index-7 >= 14 {
			value += words[index-7]
		}
		if index-15 >= 14 {
			s := sigma0(words[index-15])
			if index == 29 {
				s ^= sha256Sigma0Adjustment
			}
			value += s
		}
		words = append(words, value)
	}
	words = append(words, sigma1(words[28])+words[23]+sigma0(words[15])+words[14]+sha256Adjustment-sha256RoundConstants[14])
	words = expandSchedule(words, 31)
	working := sha256Rounds(sha256State, words, 14)
	var out [8]uint32
	for i := range out {
		out[i] = sha256Initial[i] + working[i]
	}
	return out
}

// sha256Digest 计算 v2 变体 SHA-256 摘要：(57+len(data)) 位长度、前 7 字节盐不进块。
func sha256Digest(data []byte) ([]byte, error) {
	if len(data) < 7 {
		return nil, fmt.Errorf("edge token salt must contain at least seven bytes")
	}
	length := 57 + len(data)
	padLen := ((55-length)%64 + 64) % 64
	padded := make([]byte, 0, len(data)-7+1+padLen+8)
	padded = append(padded, data[7:]...)
	padded = append(padded, 0x80)
	padded = append(padded, make([]byte, padLen)...)
	var lenBits [8]byte
	binary.BigEndian.PutUint64(lenBits[:], uint64(length)*8)
	padded = append(padded, lenBits[:]...)

	state := sha256Init(data[:7])
	for off := 0; off < len(padded); off += 64 {
		words := make([]uint32, 16, 64)
		for i := range words {
			words[i] = binary.BigEndian.Uint32(padded[off+4*i : off+4*i+4])
		}
		words = expandSchedule(words, 16)
		working := sha256Rounds(state, words, 0)
		for i := range state {
			state[i] += working[i]
		}
	}
	out := make([]byte, 32)
	for i, v := range state {
		binary.BigEndian.PutUint32(out[4*i:4*i+4], v)
	}
	return out, nil
}

// ---- CS64 ----

func wordsLE4Even(data []byte) []uint32 {
	padded := make([]byte, len(data)+(4-len(data)%4)%4)
	copy(padded, data)
	words := make([]uint32, 0, len(padded)/4+1)
	for i := 0; i < len(padded); i += 4 {
		words = append(words, binary.LittleEndian.Uint32(padded[i:i+4]))
	}
	if len(words)%2 == 1 {
		words = append(words, 0)
	}
	return words
}

func cs64Digest(data []byte, digest []byte) (uint32, uint32) {
	keyA := binary.LittleEndian.Uint32(digest[0:4]) | 1
	keyB := binary.LittleEndian.Uint32(digest[4:8]) | 1
	factors := [2]uint32{keyA * cs64Factor, keyB * cs64Factor}
	terms := make([]uint32, 0, 28)
	offsets := make([]uint32, 0, 28)
	for i := 0; i < 14; i++ {
		terms = append(terms, cs64Coefficients[i][0]*factors[i%2])
		offsets = append(offsets, cs64Coefficients[i][1])
	}
	words := wordsLE4Even(append([]byte{0}, data...))
	for i, word := range words {
		key, offFactor := keyA, uint32(0xEF0569FB)
		if i%2 == 1 {
			key, offFactor = keyB, uint32(0xC31713DB)
		}
		term := word * key
		off := rol16(word) * offFactor
		if i == 0 {
			term += cs64Adjustment[0] * factors[0]
			off += cs64Adjustment[1]
		}
		terms = append(terms, term)
		offsets = append(offsets, off)
	}
	var swapAcc, swapLast, state, acc uint32
	for i := 0; i < len(terms); i += 2 {
		x, y := terms[i], terms[i+1]
		t := rol16(offsets[i] + x)
		q := t * 0x76F50F34
		swapAcc += rol16(q)*0xC3EFEA97 + q
		t = rol16(offsets[i+1] + y)
		q = t * 0x1881CE3C
		swapLast = rol16(q)*0x35BD1EC9 + q
		swapAcc += swapLast

		state = state*keyA + x
		state = rol16(state) * 0xCF98B111
		state = rol16(state) * 0x87085B9F
		state = rol16(state) * 0x12CEB96D
		state = rol16(state) * 0x257E1D83
		acc += state
		t = state*keyB + y
		t = rol16(t) * 0xA27416F5
		t = rol16(t) * 0xD38396FF
		state = rol16(rol16(t)*0x7C932B89) * 0xBFA49F69
		acc += state
	}
	return swapLast ^ state, swapAcc ^ acc
}
