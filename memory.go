package session

import (
	"sync"
	"time"
)

type memstore struct {
	store   map[string]Sessiondata
	lock    sync.RWMutex
	memused uint64
}

func init() {
	Register("memory", memstore{})
}

func (ms memstore) Open(options string) (Store, error) {
	return memstore{store: make(map[string]Sessiondata)}, nil
}

// for session interface Get
func (ms memstore) Get(key string) Sessiondata {
	var (
		n, e time.Time
	)
	ms.lock.RLock()
	if v, ok := ms.store[key]; ok {
		ms.lock.RUnlock()
		n = time.Now()
		e = v[expiresTS].(time.Time)
		// timeout
		if n.After(e) {
			ms.lock.Lock()
			delete(ms.store, key)
			ms.lock.Unlock()
			return nil
		}
		return v
	}
	ms.lock.RUnlock()
	return nil
}

// for session interface SetStore
func (ms memstore) Set(key string, data Sessiondata, timeout int) error {
	ms.lock.Lock()
	ms.store[key] = data
	ms.lock.Unlock()
	return nil
}

// for session interface DelStore
func (ms memstore) Delete(key string) {
	ms.lock.Lock()
	delete(ms.store, key)
	ms.lock.Unlock()
}
