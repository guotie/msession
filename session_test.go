package session

import (
	"github.com/codegangsta/martini"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_Session(t *testing.T) {
	m := martini.Classic()

	m.Use(Sessions("my_session", "memory", "", "secret123"))

	m.Get("/testsession", func(res http.ResponseWriter, session Session) string {
		session.Init()
		session.SetKey("hello", "world")

		return "OK"
	})

	m.Get("/show", func(session Session) string {
		session.Init()
		if session.Get("hello") != "world" {
			t.Error("Session writing failed")
		}
		return "OK"
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/testsession", nil)
	m.ServeHTTP(res, req)

	println(res.Code)
	for k, v := range res.HeaderMap {
		print("%s: %s\n", k, v)
	}
	println(res.Body.String(), "\n")

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/show", nil)
	req2.Header.Set("Cookie", res.Header().Get("Set-Cookie"))
	m.ServeHTTP(res2, req2)
}

func Test_Flashes(t *testing.T) {
	m := martini.Classic()

	m.Use(Sessions("sid", "memory", "", "secret123"))

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
	req3.Header.Set("Cookie", res2.Header().Get("Set-Cookie"))
	m.ServeHTTP(res3, req3)
}
