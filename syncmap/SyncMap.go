package syncmap

import (
	"sync"
)

type SyncMap struct {
	sm   sync.Map
	size int
}

func (m *SyncMap) Keys() []string {
	keys := make([]string, m.size)
	i := 0
	m.sm.Range(func(k, v interface{}) bool {
		keys[i] = k.(string)
		i += 1
		return true
	})
	return keys
}

func (m *SyncMap) Load(key string) (string, bool) {
	val, ok := m.sm.Load(key)
	if !ok {
		return "", false
	}
	return val.(string), true
}

func (m *SyncMap) Store(key, value string) {
	m.sm.Store(key, value)
	m.size += 1
}

func (m *SyncMap) Delete(key string) {
	m.sm.Delete(key)
	m.size -= 1
}

func (m *SyncMap) Rename(oldkey string, newkey string) bool {
	v, ok := m.sm.Load(oldkey)
	if ok {
		m.sm.Store(newkey, v)
		m.sm.Delete(oldkey)
		return true
	} else {
		return false
	}
}

func New() *SyncMap {
	return &SyncMap{}
}
