package hotkey

import (
	"time"

	"github.com/golang/groupcache/lru"
)

type LocalCache interface {
	Add(key string, value interface{}, ttl uint32)
	Get(key string) (interface{}, bool)
	Remove(key string)
}

type item struct {
	ttl uint32
	val interface{}
}

func NewLocalCache(cap int) LocalCache {
	return &localCache{cache: lru.New(cap), startTime: time.Now().UnixNano() / int64(time.Millisecond)}
}

type localCache struct {
	cache *lru.Cache
	// 减少item ttl开销
	startTime int64
}

// Add add key value with TTL to local cache
func (l *localCache) Add(key string, value interface{}, ttl uint32) {
	item := &item{ttl + uint32(time.Now().UnixNano()/int64(time.Millisecond)-l.startTime), value}
	l.cache.Add(key, item)
}

func (l *localCache) Get(key string) (interface{}, bool) {
	if v, ok := l.cache.Get(key); ok {
		val := v.(*item)
		if int64(val.ttl) > (time.Now().UnixNano()/int64(time.Millisecond) - l.startTime) {
			return val.val, true
		}
		l.cache.Remove(key)
	}
	return "", false
}

func (l *localCache) Remove(key string) {
	l.cache.Remove(key)
}
