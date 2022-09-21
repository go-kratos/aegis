package hotkey

import (
	"regexp"
	"sync"

	"github.com/go-kratos/aegis/pkg/minheap"
)

type CacheRuleConfig struct {
	Mode  string `toml:"match_mode"`
	Value string `toml:"match_value"`
	TTLMs uint32 `toml:"ttl_ms"`
}
type HotkeyCache interface {
	Add(key string, incr uint32)
	AddWithValue(key string, value interface{}, incr uint32)
	DelCache(key string)
	List() minheap.Nodes
	Get(key string) (interface{}, bool)
}
type Option struct {
	HotKeyCnt     int
	LocalCacheCnt int
	AutoCache     bool
	CacheMs       int
	WhileList     []*CacheRuleConfig
	BlackList     []*CacheRuleConfig
}

var (
	ruleTypeKey     = "key"
	ruleTypePattern = "pattern"
)

type cacheRule struct {
	value  string
	regexp *regexp.Regexp
	ttl    uint32
}

type hotKeyWithCache struct {
	topk       *TopK
	mutex      sync.Mutex
	option     *Option
	localCache LocalCache
	whilelist  []*cacheRule
	blacklist  []*cacheRule
}

func NewHotkey(option *Option) HotkeyCache {
	h := &hotKeyWithCache{option: option}
	if option.HotKeyCnt > 0 {
		h.topk = NewTopk(uint32(option.HotKeyCnt), 1024, 4, 0.925)
	}
	if len(h.option.WhileList) > 0 {
		h.whilelist = h.initCacheRules(h.option.WhileList)
	}
	if len(h.option.BlackList) > 0 {
		h.blacklist = h.initCacheRules(h.option.BlackList)
	}
	if h.option.AutoCache || len(h.whilelist) > 0 {
		h.localCache = NewLocalCache(int(h.option.LocalCacheCnt))
	}
	return h
}

func (h *hotKeyWithCache) initCacheRules(rules []*CacheRuleConfig) []*cacheRule {
	list := make([]*cacheRule, 0, len(rules))
	for _, rule := range rules {
		ttl := rule.TTLMs
		if ttl == 0 {
			ttl = uint32(h.option.CacheMs)
		}
		cacheRule := &cacheRule{ttl: ttl}
		if rule.Mode == ruleTypeKey {
			cacheRule.value = rule.Value
		} else if rule.Mode == ruleTypePattern {
			regexp, err := regexp.Compile(rule.Value)
			if err != nil {
				continue
			}
			cacheRule.regexp = regexp
		} else {
			continue
		}
		list = append(list, cacheRule)
	}
	return list
}

func (h *hotKeyWithCache) inBlacklist(key string) bool {
	if len(h.blacklist) == 0 {
		return false
	}
	for _, b := range h.blacklist {
		if b.value == key || b.regexp.Match([]byte(key)) {
			return true
		}
	}
	return false
}

func (h *hotKeyWithCache) inWhitelist(key string) (uint32, bool) {
	if len(h.whilelist) == 0 {
		return 0, false
	}
	for _, b := range h.whilelist {
		if b.value == key || b.regexp.Match([]byte(key)) {
			return b.ttl, true
		}
	}
	return 0, false
}

func (h *hotKeyWithCache) Add(key string, incr uint32) {
	if h.topk == nil {
		return
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.topk.Add(key, incr)
}

func (h *hotKeyWithCache) AddWithValue(key string, value interface{}, incr uint32) {
	if h.topk == nil && h.localCache == nil {
		return
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.topk != nil {
		expelled, added := h.topk.Add(key, incr)
		if len(expelled) > 0 && h.localCache != nil {
			h.localCache.Remove(expelled)
		}
		if h.option.AutoCache && added {
			if !h.inBlacklist(key) {
				h.localCache.Add(key, value, uint32(h.option.CacheMs))
			}
			return
		}
	}
	if ttl, ok := h.inWhitelist(key); ok {
		h.localCache.Add(key, value, ttl)
	}
}

func (h *hotKeyWithCache) DelCache(key string) {
	if h.localCache == nil {
		return
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.localCache.Remove(key)
}

func (h *hotKeyWithCache) Get(key string) (interface{}, bool) {
	if h.localCache == nil {
		return "", false
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if v, ok := h.localCache.Get(key); ok {
		return v, true
	}
	return "", false
}

func (h *hotKeyWithCache) List() minheap.Nodes {
	if h.topk == nil {
		return nil
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.topk.List()
}
