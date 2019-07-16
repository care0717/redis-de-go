package main

import "sync"

type SyncMap struct {
	sm sync.Map
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
}

func (m *SyncMap) Delete(key string) {
	m.sm.Delete(key)
}

func (m *SyncMap) Rename(oldkey string, newkey string) bool {
	v, ok := memory.Load(oldkey)
	if ok {
		memory.Store(newkey, v)
		memory.Delete(oldkey)
		return true
	} else {
		return false
	}
}

func New() *SyncMap {
	return &SyncMap{}
}
