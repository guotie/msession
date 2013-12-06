package session

import (
	"github.com/codegangsta/martini"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_SessionTO(t *testing.T) {
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

func Test_SessionRefresh(t *testing.T) {
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

func Test_SessionClear(t *testing.T) {
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

func Test_Flashes(t *testing.T) {
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
