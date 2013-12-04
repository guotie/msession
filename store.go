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
