package session

import (
	"fmt"
)

var _ = fmt.Printf

type Store interface {
	Open(options string) (Store, error)
	Get(key string) Sessiondata
	Set(key string, data Sessiondata, timeout int) error
	Delete(string)

	// For memory store, return true, otherwise false
	// This is used by session to distinguaish memory store and others
	// For memory store, we set/delete key direct from memory, so
	// it is necessary to lock the memory mutex.
	Memory() bool
}

var stores = make(map[string]Store)

func Register(name string, store Store) {
	if store == nil {
		panic("session: Register store is nil")
	}
	if _, dup := stores[name]; dup {
		panic("session: Register called twice for store " + name)
	}

	stores[name] = store
}

func Open(name, options string) (Store, error) {
	s := stores[name]
	if s == nil {
		return nil, fmt.Errorf("session: No such session store type: %s", name)
	}

	return s.Open(options)
}
