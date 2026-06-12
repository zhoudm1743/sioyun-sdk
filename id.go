// Package id 提供时序 ID 生成器（移植自 backend/framework/id）。
//
// 规格：
//   - 长度：16 字符（固定）
//   - 字符集：Crockford Base32 小写（去掉 i l o u，避免视觉混淆）
//   - 结构：[10 chars | 50-bit 毫秒时间戳] [6 chars | 30-bit 单调序列]
//   - 字典序 ≡ 时间序：可直接在数据库 ORDER BY id ASC/DESC 按创建时间排序
//   - 并发安全：全局 Mutex 保护，同一毫秒内序列单调递增
//   - 时钟回拨安全：lastMs 只增不减，NTP 调整后 ID 仍单调递增
package sioyun

import (
	"crypto/rand"
	"encoding/binary"
	"sync"
	"time"
)

const charset = "0123456789abcdefghjkmnpqrstvwxyz"

var (
	idMu     sync.Mutex
	idLastMs int64
	idSeq    uint32
	idNowFn  = func() int64 { return time.Now().UnixMilli() }
)

// newID 生成一个 16 字符的时序 ID，并发安全。
func newID() string {
	idMu.Lock()
	now := idNowFn()
	if now > idLastMs {
		idLastMs = now
		idSeq = randUint30()
	} else {
		idSeq = (idSeq + 1) & 0x3FFFFFFF
		if idSeq == 0 {
			idLastMs++
			idSeq = randUint30()
		}
	}
	ts := uint64(idLastMs) & ((uint64(1) << 50) - 1)
	r := idSeq
	idMu.Unlock()
	return encode50_30(ts, r)
}

func encode50_30(ts50 uint64, r30 uint32) string {
	var b [16]byte
	for i := 9; i >= 0; i-- {
		b[i] = charset[ts50&0x1F]
		ts50 >>= 5
	}
	for i := 15; i >= 10; i-- {
		b[i] = charset[r30&0x1F]
		r30 >>= 5
	}
	return string(b[:])
}

func randUint30() uint32 {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return uint32(time.Now().UnixNano()) & 0x3FFFFFFF
	}
	return binary.BigEndian.Uint32(b[:]) & 0x3FFFFFFF
}
