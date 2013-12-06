package session

import (
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"time"
)

type redisstore struct {
	pool    *redis.Pool
	memused uint64
}

const (
	defaultAddr     = "localhost:9368"
	defaultNetwork  = "tcp"
	defaultPoolSize = 10
)

var ()

func init() {
	Register("redis", redisstore{})
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
		println("unmarshal", options, "failed:", err)
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
	val, ok := rs.pool.Get().Do("GET", key)
	if ok != nil {
		return nil
	}
	return val.(Sessiondata)
}

// for session interface SetStore
func (rs redisstore) Set(key string, data Sessiondata, timeout int) error {

	return nil
}

// for session interface DelStore
func (rs redisstore) Delete(key string) {
	rs.pool.Get().Do("DELETE", key)
}

func (rs redisstore) Memory() bool {
	return false
}