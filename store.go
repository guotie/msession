package session

import (
	"fmt"
)

var _ = fmt.Printf

type Store interface {
	Open()
	Get(key interface{}) *sessiondata
	Set(key interface{}, data *sessiondata, timeout int) error
	Delete(interface{})
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
	return nil, nil
}
