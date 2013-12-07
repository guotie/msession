package session

import (
	"encoding/gob"
	//"github.com/garyburd/redigo/redis"
	"bytes"
	"testing"
	"time"
)

type TM map[interface{}]interface{}

func enc(m TM) ([]byte, error) {
	buf := new(bytes.Buffer)
	encd := gob.NewEncoder(buf)
	if err := encd.Encode(m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func dec(buf []byte, dst TM) error {
	decd := gob.NewDecoder(bytes.NewBuffer(buf))
	if err := decd.Decode(dst); err != nil {
		return err
	}

	return nil
}

func Test_Conn(t *testing.T) {
	pool := createPool("")
	conn := pool.Get()
	m := TM{"str": "abcdefg",
		1:         100,
		true:      "true",
		"false":   false,
		"expires": time.Now(),
	}

	buf, err := enc(m)
	if err != nil {
		panic(err)
	}
	_, err = conn.Do("SET", "id1", buf)
	if err != nil {
		t.Error("set map failed!")
	} else {
		println("redis set success.")
	}
}

/*
import (
	"github.com/codegangsta/martini"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_RedisSessionTO(t *testing.T) {
	m := martini.Classic()

	m.Use(Sessions("my_session", "redis", "", "secret123"))

	m.Get("/testsession", func(res http.ResponseWriter, session Session) string {
		session.Init()
		session.Create(5, nil)
		session.SetKey("hello", "world")

		return "OK"
	})

	m.Get("/show", func(session Session) string {
		session.Init()
		if session.Get("hello") != "world" {
			t.Error("Session write failed")
		}
		return "OK"
	})

	m.Get("/show2", func(session Session) string {
		session.Init()
		if session.Get("hello") == "world" {
			t.Error("Session timeout failed")
		}
		return "OK"
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/testsession", nil)
	m.ServeHTTP(res, req)

	time.Sleep(time.Second * time.Duration(1))

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/show", nil)
	req2.Header.Set("Cookie", res.Header().Get("Set-Cookie"))
	m.ServeHTTP(res2, req2)

	time.Sleep(time.Second * time.Duration(6))

	res3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/show2", nil)
	req3.Header.Set("Cookie", res.Header().Get("Set-Cookie"))
	m.ServeHTTP(res3, req3)
}

func Test_RedisSessionRefresh(t *testing.T) {
	m := martini.Classic()

	m.Use(Sessions("sid", "redis", "", "secret123"))

	m.Get("/testsession", func(res http.ResponseWriter, session Session) string {
		session.Init()
		session.Create(5, nil)
		session.SetKey("hello", "world")

		return "OK"
	})

	m.Get("/show", func(session Session) string {
		session.Init()
		if session.Get("hello") != "world" {
			t.Error("Session write failed")
		}
		return "OK"
	})

	m.Get("/show2", func(sess Session) string {
		sess.Init()
		if sess.Get("hello") == "world" {
			s := sess.(*session)
			tm := (s.data[expiresTS]).(time.Time)
			t.Error(tm)
			t.Error("Session refresh timeout failed")
		}
		return "OK"
	})

	m.Get("/refresh", func(session Session) string {
		session.Init()
		session.Refresh(time.Duration(5) * time.Second)
		return "OK"
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/testsession", nil)
	m.ServeHTTP(res, req)

	time.Sleep(time.Second * time.Duration(1))

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/show", nil)
	req2.Header.Set("Cookie", res.Header().Get("Set-Cookie"))
	m.ServeHTTP(res2, req2)

	res3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/refresh", nil)
	req3.Header.Set("Cookie", res.Header().Get("Set-Cookie"))
	m.ServeHTTP(res3, req3)

	println(res.Header().Get("Set-Cookie"))
	println(res3.Header().Get("Set-Cookie"))
	time.Sleep(time.Second * time.Duration(6))

	res4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", "/show", nil)
	req4.Header.Set("Cookie", res3.Header().Get("Set-Cookie"))
	m.ServeHTTP(res4, req4)

	time.Sleep(time.Second * time.Duration(6))
	res5 := httptest.NewRecorder()
	req5, _ := http.NewRequest("GET", "/show2", nil)
	req5.Header.Set("Cookie", res3.Header().Get("Set-Cookie"))
	m.ServeHTTP(res5, req5)
}

func Test_RedisSessionClear(t *testing.T) {
	m := martini.Classic()

	m.Use(Sessions("sid", "redis", "", "secret123"))

	m.Get("/testsession", func(res http.ResponseWriter, session Session) string {
		session.Init()
		session.Create(5, nil)
		session.SetKey("hello", "world")
		session.SetKey("who", "guotie")

		return "OK"
	})

	m.Get("/delkey", func(res http.ResponseWriter, session Session) string {
		session.Init()
		session.DelKey("hello")

		return "OK"
	})

	m.Get("/getdelkey", func(res http.ResponseWriter, session Session) string {
		session.Init()
		if session.Get("hello") == "world" {
			t.Error("Session delkey failed")
		}
		return "OK"
	})

	m.Get("/clear", func(res http.ResponseWriter, session Session) string {
		session.Init()
		session.Clear(res)
		return "OK"
	})

	m.Get("/show", func(session Session) string {
		if session.Init() == true {
			//t.Error("session clear failed!")
			print("session exist\n")
		}
		if session.Get("hello") == "world" {
			t.Error("Session clear failed")
		}
		return "OK"
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/testsession", nil)
	m.ServeHTTP(res, req)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/delkey", nil)
	req2.Header.Set("Cookie", res.Header().Get("Set-Cookie"))
	m.ServeHTTP(res2, req2)

	res3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/getdelkey", nil)
	req3.Header.Set("Cookie", res.Header().Get("Set-Cookie"))
	m.ServeHTTP(res3, req3)

	res4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", "/clear", nil)
	req4.Header.Set("Cookie", res.Header().Get("Set-Cookie"))
	m.ServeHTTP(res4, req4)

	time.Sleep(time.Second * time.Duration(1))

	res5 := httptest.NewRecorder()
	req5, _ := http.NewRequest("GET", "/show", nil)
	req5.Header.Set("Cookie", res.Header().Get("Set-Cookie"))
	m.ServeHTTP(res5, req5)
}

func Test_RedisFlashes(t *testing.T) {
	m := martini.Classic()

	m.Use(Sessions("sid", "redis", "", "secret123"))

	m.Get("/set", func(session Session) string {
		session.Init()
		session.AddFlash("hello world")
		return "OK"
	})

	m.Get("/show", func(session Session) string {
		session.Init()
		l := len(session.Flashes())
		if l != 1 {
			t.Error("Flashes count does not equal 1. Equals ", l)
		}
		return "OK"
	})

	m.Get("/showagain", func(session Session) string {
		session.Init()
		l := len(session.Flashes())
		if l != 0 {
			t.Error("flashes count is not 0 after reading. Equals ", l)
		}
		return "OK"
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/set", nil)
	m.ServeHTTP(res, req)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/show", nil)
	req2.Header.Set("Cookie", res.Header().Get("Set-Cookie"))
	m.ServeHTTP(res2, req2)

	res3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/showagain", nil)
	req3.Header.Set("Cookie", res.Header().Get("Set-Cookie"))
	m.ServeHTTP(res3, req3)
}
*/
