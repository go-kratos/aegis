package syncmap

import "sync"

type SyncMap[K comparable, V any] struct {
	sync.Map
}

func (m *SyncMap[K, V]) Load(key K) (V, bool) {
	value, ok := m.Map.Load(key)
	if !ok {
		var zero V
		return zero, false
	}
	return value.(V), true
}

func (m *SyncMap[K, V]) Store(key K, value V) {
	m.Map.Store(key, value)
}

func (m *SyncMap[K, V]) Delete(key K) {
	m.Map.Delete(key)
}

func (m *SyncMap[K, V]) Range(f func(key K, value V) bool) {
	m.Map.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	loaded, ok := m.Map.LoadOrStore(key, value)
	if !ok {
		return value, false
	}
	return loaded.(V), true
}
