package hotkey

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func benchmarkHotkey(b *testing.B, autoCache bool, writePercent float64, whilelist ...*CacheRuleConfig) {
	option := &Option{
		HotKeyCnt:     100,
		LocalCacheCnt: 100,
		AutoCache:     autoCache,
		CacheMs:       100,
		WhileList:     whilelist,
	}

	h := NewHotkey(option)
	random := rand.New(rand.NewSource(time.Now().Unix()))
	zipf := rand.NewZipf(rand.New(rand.NewSource(time.Now().Unix())), 2, 2, 1000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := strconv.FormatUint(zipf.Uint64(), 10)
			if random.Float64() < writePercent {
				h.AddWithValue(key, key, 1)
			} else {
				h.Get(key)
			}
		}
	})

}

func BenchmarkHotkeyAutoCacheWrite1_100(b *testing.B) {
	benchmarkHotkey(b, true, 0.01)
}
func BenchmarkHotkeyAutoCacheWrite10_100(b *testing.B) {
	benchmarkHotkey(b, true, 0.1)
}

func BenchmarkHotkeyAutoCacheWrite50_100(b *testing.B) {
	benchmarkHotkey(b, true, 0.5)
}

func BenchmarkHotkeyAutoCacheWrite100_100(b *testing.B) {
	benchmarkHotkey(b, true, 1)
}

func BenchmarkHotkeyWhilelist1Write10_100(b *testing.B) {
	var cacheRules []*CacheRuleConfig
	cacheRules = append(cacheRules, &CacheRuleConfig{Mode: "pattern", Value: "[0-9]{1,3}", TTLMs: 100})
	benchmarkHotkey(b, false, 0.1, cacheRules...)
}

func BenchmarkHotkeyWhilelist5Write10_100(b *testing.B) {
	var cacheRules []*CacheRuleConfig
	cacheRules = append(cacheRules, &CacheRuleConfig{Mode: "pattern", Value: "[0-1]{1,3}", TTLMs: 100})
	cacheRules = append(cacheRules, &CacheRuleConfig{Mode: "pattern", Value: "[2-3]{1,3}", TTLMs: 100})
	cacheRules = append(cacheRules, &CacheRuleConfig{Mode: "pattern", Value: "[4-5]{1,3}", TTLMs: 100})
	cacheRules = append(cacheRules, &CacheRuleConfig{Mode: "pattern", Value: ".*", TTLMs: 100})
	benchmarkHotkey(b, false, 0.1, cacheRules...)
}
func TestOnlyWhileList(t *testing.T) {
	var cacheRules []*CacheRuleConfig
	cacheRules = append(cacheRules, &CacheRuleConfig{Mode: "pattern", Value: "^1[0-9]{2}", TTLMs: 100})
	option := &Option{
		LocalCacheCnt: 100,
		AutoCache:     false,
		CacheMs:       100,
		WhileList:     cacheRules,
	}

	h := NewHotkey(option)
	for i := 0; i < 100; i++ {
		key := strconv.FormatInt(int64(i), 10)
		h.AddWithValue(key, key, 1)
		_, ok := h.Get(key)
		assert.False(t, ok, key)
	}
	for i := 100; i < 200; i++ {
		key := strconv.FormatInt(int64(i), 10)
		h.AddWithValue(key, key, 1)
		_, ok := h.Get(key)
		assert.True(t, ok, key)
	}
	hots := h.List()
	assert.Equal(t, 0, len(hots))
}
func TestHotkeyWhilelist(t *testing.T) {
	var cacheRules []*CacheRuleConfig
	cacheRules = append(cacheRules, &CacheRuleConfig{Mode: "pattern", Value: "^1[0-9]{1,2}", TTLMs: 100})
	option := &Option{
		HotKeyCnt:     100,
		LocalCacheCnt: 100,
		AutoCache:     false,
		CacheMs:       100,
		WhileList:     cacheRules,
	}

	h := NewHotkey(option)
	for i := 100; i < 200; i++ {
		key := strconv.FormatInt(int64(i), 10)
		h.AddWithValue(key, key, 1)
		_, ok := h.Get(key)
		assert.True(t, ok, key)
	}
	for i := 200; i < 300; i++ {
		key := strconv.FormatInt(int64(i), 10)
		h.AddWithValue(key, key, 1)
		_, ok := h.Get(key)
		assert.False(t, ok, key)
	}
}

func TestHotkeyBlacklist(t *testing.T) {
	var cacheRules []*CacheRuleConfig
	cacheRules = append(cacheRules, &CacheRuleConfig{Mode: "pattern", Value: "^2$", TTLMs: 100})
	cacheRules = append(cacheRules, &CacheRuleConfig{Mode: "pattern", Value: "^3$", TTLMs: 100})

	option := &Option{
		HotKeyCnt:     100,
		LocalCacheCnt: 100,
		AutoCache:     true,
		CacheMs:       100,
		BlackList:     cacheRules,
	}

	h := NewHotkey(option)
	zipf := rand.NewZipf(rand.New(rand.NewSource(time.Now().Unix())), 2, 2, 1000)
	for i := 0; i < 100000; i++ {
		key := strconv.FormatInt(int64(zipf.Uint64()), 10)
		h.AddWithValue(key, key, 1)
	}
	for i := 0; i < 10; i++ {
		key := strconv.FormatInt(int64(i), 10)
		_, ok := h.Get(key)
		if i == 2 || i == 3 {
			assert.False(t, ok)
		} else {
			assert.True(t, ok)
		}
	}
}
