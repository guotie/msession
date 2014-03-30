package session

import (
	"encoding/hex"
	"fmt"
	"github.com/codegangsta/martini"
	"github.com/streadway/simpleuuid"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	warnFormat  = "[sessions] WARN: %s\n"
	errorFormat = "[sessions] ERROR: %s\n"
)

// Session stores the values and optional configuration for a session.
type Session interface {
	// Init session by cookie name, fetch data
	Init() bool

	// Get returns the session value associated to the given key.
	Get(key interface{}) interface{}

	// Create a new session ID with sessiondata
	Create(age int, l *log.Logger)

	// Set sets the session value associated to the given key.
	SetKey(key interface{}, val interface{})

	// Set sessiondata back to store
	SetStore() error

	// Delete the key/value of session data
	DelKey(key interface{})

	// Delete the session data from store
	DelStore()

	// Save is to the client, usualy browsers
	Save(res http.ResponseWriter)

	// Refresh session's expire time by add duration t
	Refresh(t time.Duration)
	// Refresh session's expire to time t
	RefreshTO(t time.Time)

	// Clear cookie, by set cookie's expire to now
	Clear(res http.ResponseWriter)

	// AddFlash adds a flash message to the session.
	AddFlash(value interface{})

	// Flashes returns a slice of flash messages from the session.
	Flashes() []interface{}
}

var (
	_           = fmt.Printf
	store       Store
	sessionname string = "sid"
	secretKey   []byte
	maxAge      int           = 30 * 86400
	maxDurtion  time.Duration = time.Duration(maxAge) * time.Second
	httpOnly    bool          = true
	secure      bool
)

// Sessions is a Middleware that maps a session.Session service into the Martini
// handler chain.
// Sessions can use a number of storage solutions with the given store options.
// store: memory, redis, memcache, etc
// options: usally json string to open store
func Sessions(name string,
	storetype string,
	dsn string,
	secret string) martini.Handler {
	var err error

	store, err = Open(storetype, dsn)
	if err != nil {
		panic(err)
	}
	if name != "" {
		sessionname = name
	}
	if secret == "" {
		panic("sessions: secret should not be empty!\n")
	}
	secretKey = []byte(secret)

	return func(res http.ResponseWriter, r *http.Request, c martini.Context,
		l *log.Logger) {
		// Map to the Session interface
		s := &session{}
		s.cookie, _ = r.Cookie(name)
		c.MapTo(s, (*Session)(nil))

		rw := res.(martini.ResponseWriter)
		rw.Before(func(martini.ResponseWriter) {
			if s.shouldset {
				check(s.SetStore(), l)
			}
			if s.shouldsave {
				s.Save(res)
			}
		})
	}
}

func check(err error, l *log.Logger) {
	if err != nil {
		l.Printf(errorFormat, err)
	}
}

/*
 *-------------------------global session getting/setting-----------------------
 */
func MaxAge() int {
	return maxAge
}

func SetMaxAge(age int) {
	maxAge = age
	maxDurtion = time.Duration(maxAge) * time.Second
}

func HttpOnly() bool {
	return httpOnly
}

func SetHttpOnly(http bool) {
	httpOnly = http
}

func Secure() bool {
	return secure
}

func SetSecure(s bool) {
	secure = s
}

/*
 *---------------------------session implement----------------------------------
 */
type Sessiondata map[interface{}]interface{}

type session struct {
	key    string
	cookie *http.Cookie
	data   Sessiondata

	// status of the session
	// true: initialed, and get data successfully, otherwise false
	status bool
	// send set-cookie to browser
	shouldsave bool
	// set data back to store
	shouldset bool
	// send set-cookie to browser to clear cookie
	clear bool
}

const (
	flashesKey = "_flash"
	expiresTS  = "_expires"
)

// Returns true if a Session pulled from signed cookie else false
func (s *session) Init() bool {
	cookie := s.cookie
	if cookie == nil {
		return false
	}

	// Separate the data from the signature.
	hyphen := strings.Index(cookie.Value, "-")
	if hyphen == -1 || hyphen >= len(cookie.Value)-1 {
		return false
	}
	sig, data := cookie.Value[:hyphen], cookie.Value[hyphen+1:]

	// Verify the signature.
	if !Verify(data, sig) {
		return false
	}

	s.key = data
	s.data = store.Get(data)
	s.status = true

	return true
}

// Get returns the session value associated to the given key.
func (s *session) Get(key interface{}) interface{} {
	if !s.status {
		return nil
	}
	if s.data == nil {
		return nil
	}

	if store.Memory() {
		st := store.(memstore)
		st.lock.RLock()
		val := s.data[key]
		st.lock.RUnlock()
		return val
	}

	return s.data[key]
}

// Create a new session ID with sessiondata
// if age is greater than zero, we will use this age overwrite the global maxAge
func (s *session) Create(age int, l *log.Logger) {
	if s.data != nil || s.cookie != nil {
		if l != nil {
			l.Println(warnFormat, "Overwrite exist session "+s.key)
		}
	}

	uuid, err := simpleuuid.NewTime(time.Now())
	if err != nil {
		panic(err) // I don't think this can actually happen.
	}

	s.key = hex.EncodeToString(uuid[0:16])
	s.data = make(Sessiondata)
	if age > 0 {
		s.data[expiresTS] = time.Now().Add(time.Duration(age) * time.Second)
	} else {
		s.data[expiresTS] = time.Now().Add(maxDurtion)
	}

	s.shouldset = true
	s.shouldsave = true
	s.status = true
}

// Set sets the session value associated to the given key.
func (s *session) SetKey(key interface{}, val interface{}) {
	if s.data == nil || !s.status {
		s.Create(0, nil)
	}
	if store.Memory() {
		st := store.(memstore)
		st.lock.Lock()
		s.data[key] = val
		st.lock.Unlock()
	} else {
		s.data[key] = val
		s.shouldset = true
	}
}

// set session data back to store
func (s *session) SetStore() error {
	s.shouldset = false
	if store.Memory() {
		return store.Set(s.key, s.data, 0)
	}

	now := time.Now()
	delta := s.data[expiresTS].(time.Time).Sub(now)
	age := int(delta / time.Second)
	return store.Set(s.key, s.data, age)
}

// Delete the key/value of session data
func (s *session) DelKey(key interface{}) {
	if s.data == nil {
		return
	}
	if store.Memory() {
		st := store.(memstore)

		st.lock.Lock()
		delete(s.data, key)
		st.lock.Unlock()
	} else {
		delete(s.data, key)
		s.shouldset = true
	}
}

// Delete the session data from store
func (s *session) DelStore() {
	if s.data != nil {
		s.data = nil
		store.Delete(s.key)
		s.shouldset = false
	}
}

// Save is to the client, usualy browsers
func (s *session) Save(res http.ResponseWriter) {
	s.shouldsave = false
	cookie := &http.Cookie{
		Name:     sessionname,
		Value:    Sign(s.key) + "-" + s.key,
		Path:     "/",
		HttpOnly: httpOnly,
		Secure:   secure,
		Expires:  s.data[expiresTS].(time.Time).UTC(),
	}
	http.SetCookie(res, cookie)
	return
}

// clear this cookie, by set Expires to now
func (s *session) Clear(res http.ResponseWriter) {
	s.DelStore()
	s.shouldsave = false

	cookie := &http.Cookie{
		Name:     sessionname,
		Value:    Sign(s.key) + "-" + s.key,
		Path:     "/",
		HttpOnly: httpOnly,
		Secure:   secure,
		Expires:  time.Now().UTC(),
	}
	http.SetCookie(res, cookie)
	return
}

// Refresh session's expire time by add duration t
// param t is duration
func (s *session) Refresh(t time.Duration) {
	v := s.data[expiresTS].(time.Time).Add(t)
	n := time.Now()

	if store.Memory() {
		st := store.(memstore)
		st.lock.Lock()
		s.data[expiresTS] = v
		tmr := s.data["_tmr"].(*time.Timer)
		if v.After(n) {
			tmr.Reset(v.Sub(time.Now()))
		}
		st.lock.Unlock()
	} else {
		s.data[expiresTS] = v
		s.shouldset = true
	}
	s.shouldsave = true
}

// Refresh session's expire to time t
// param t is absolute time
func (s *session) RefreshTO(t time.Time) {
	t1 := s.data[expiresTS].(time.Time)
	s.Refresh(t.Sub(t1))
}

func (s *session) AddFlash(value interface{}) {
	var flashes []interface{} = make([]interface{}, 0)

	if v, ok := s.data[flashesKey]; ok {
		flashes = v.([]interface{})
	}
	if s.data == nil {
		s.Create(0, nil)
	}
	s.data[flashesKey] = append(flashes, value)
	s.shouldset = true
}

func (s *session) Flashes() []interface{} {
	var flashes []interface{}

	if v, ok := s.data[flashesKey]; ok {
		// Drop the flashes and return it.
		delete(s.data, flashesKey)
		flashes = v.([]interface{})
		s.shouldset = true
	}

	return flashes
}
