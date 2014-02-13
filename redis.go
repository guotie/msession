package session

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"time"
)

type redisstore struct {
	pool    *redis.Pool
	memused uint64
}

const (
	defaultAddr     = "localhost:6379"
	defaultNetwork  = "tcp"
	defaultPoolSize = 10
)

func init() {
	Register("redis", redisstore{})
	gob.Register(time.Time{})
	gob.Register([]interface{}{})
}

// options sample:
//   `{  "addr": "127.0.0.1:6389",
//       "network":"tcp",
//       "db": 0,
//       "password": "",
//       "pools": 5
//    }`
func createPool(options string) *redis.Pool {
	var config struct {
		Addr     string
		Db       int
		Network  string
		Password string
		Pools    int
	}

	err := json.Unmarshal([]byte(options), &config)
	if err != nil {
		//println("unmarshal failed:", err.Error())
		config.Addr = defaultAddr
		config.Network = defaultNetwork
		config.Pools = defaultPoolSize
	}

	if config.Pools < 0 {
		config.Pools = defaultPoolSize
	}

	pool := &redis.Pool{
		MaxIdle:     config.Pools,
		IdleTimeout: 600 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(config.Network, config.Addr)
			if err != nil {
				panic(err)
				return nil, err
			}
			if config.Password != "" {
				if _, err := c.Do("AUTH", config.Password); err != nil {
					c.Close()
					panic(err)
					return nil, err
				}
			}

			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	return pool
}

// Open redis connection
func (rs redisstore) Open(options string) (Store, error) {
	return redisstore{pool: createPool(options)}, nil
}

// for session interface Get
func (rs redisstore) Get(key string) Sessiondata {
	val, ok := rs.pool.Get().Do("HGET", "sessions", key)
	if ok != nil || val == nil {
		return nil
	}
	data, err := deserialize(val.([]byte))
	if err != nil {
		fmt.Printf("redis GET failed: deserialize: %s\n", err.Error())
	}
	exp := data[expiresTS].(time.Time)
	n := time.Now()
	if n.After(exp) {
		rs.pool.Get().Do("HDEL", "sessions", key)
		return nil
	}

	return data
}

// for session interface SetStore
func (rs redisstore) Set(key string, data Sessiondata, timeout int) error {
	buf, err := serialize(data)
	if err != nil {
		println(err.Error())
		return err
	}

	_, err = rs.pool.Get().Do("HSET", "sessions", key, buf)
	if err != nil {
		return err
	}
	return nil
}

// for session interface DelStore
func (rs redisstore) Delete(key string) {
	rs.pool.Get().Do("HDEL", "sessions", key)
}

func (rs redisstore) Memory() bool {
	return false
}

func serialize(data Sessiondata) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func deserialize(src []byte) (Sessiondata, error) {
	dst := make(Sessiondata)
	dec := gob.NewDecoder(bytes.NewBuffer(src))
	if err := dec.Decode(&dst); err != nil {
		return dst, err
	}
	return dst, nil
}
