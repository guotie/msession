package session

import (
	"encoding/hex"
	"github.com/codegangsta/martini"
	"github.com/streadway/simpleuuid"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	warnFormat  = "[sessions] WARN! %s\n"
	errorFormat = "[sessions] ERROR! %s\n"
)

// Session stores the values and optional configuration for a session.
type Session interface {
	// Init session by cookie name, fetch data
	Init()

	// Fini is called at the handler flow end, save or set, delete or clear
	Fini()

	// Get returns the session value associated to the given key.
	Get(key interface{}) interface{}

	// Create a new session ID with sessiondata
	Create()

	// Set sets the session value associated to the given key.
	SetKey(key interface{}, val interface{})

	// Set sessiondata back to store
	SetStore() error

	// Save is to the client, usualy browsers
	Save(r *http.Request) error

	// Clear cookie, by set cookie's expire to now
	Clear(r *http.Request) error

	// AddFlash adds a flash message to the session.
	AddFlash(value interface{})

	// Flashes returns a slice of flash messages from the session.
	Flashes() []interface{}
}

var (
	store       Store
	sessionname string = "sid"
	secretKey   []byte
	maxAge      int = 30 * 86400
	maxDurtion  time.Duration
	httpOnly    bool
	secure      bool
)

// Sessions is a Middleware that maps a session.Session service into the Martini
// handler chain.
// Sessions can use a number of storage solutions with the given store options.
// store: memory, redis, memcache, etc
// options: usally json string to open store
func Sessions(name string, storetype string,
	dsn string, secret string) martini.Handler {
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
		var (
			s      *session
			err    error
			cookie *http.Cookie
		)

		s = &session{}
		cookie, err = r.Cookie(name)
		if err == nil {
			s.cookie = cookie
		}

		// Map to the Session interface
		c.MapTo(s, (*Session)(nil))

		c.Next()
		if s.shouldsave {
			check(s.Save(), l)
		}
		if s.shouldset {
			check(s.SetStore(), l)
		}
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
type sessiondata map[interface{}]interface{}

type session struct {
	key    string
	cookie *http.Cookie
	data   sessiondata

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

// Returns a Session pulled from signed cookie.
func (s *session) Init() {
	cookie := s.cookie

	// Separate the data from the signature.
	hyphen := strings.Index(cookie.Value, "-")
	if hyphen == -1 || hyphen >= len(cookie.Value)-1 {
		return
	}
	sig, data := cookie.Value[:hyphen], cookie.Value[hyphen+1:]

	// Verify the signature.
	if !Verify(data, sig) {
		return
	}

	s.key = data
	s.data = store.Get(data)

	return
}

// Get returns the session value associated to the given key.
func (s *session) Get(key interface{}) interface{} {
	if s.data == nil {
		return nil
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
	s.data = make(sessiondata)
	if age > 0 {
		s.data[expiresTS] = time.Now().Add(maxDurtion)
	} else {
		s.data[expiresTS] = time.Now().Add(time.Duration(age) * time.Second)
	}

	s.shouldsave = true
}

// Set sets the session value associated to the given key.
func (s *session) SetKey(key interface{}, val interface{}) {
	if s.data == nil {
		s.Create(0, nil)
	}
	s.data[key] = val
	s.shouldset = true
}

// set session data back to store
func (s *session) SetStore() error {
	s.shouldset = false
	return store.Set(s.key, s.data, maxAge)
}

// Delete the key/value of session data
func (s *session) DelKey(key interface{}) {
	if s.data == nil {
		return
	}
	delete(s.data, key)
	s.shouldset = true
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
func (s *session) Save() error {
	s.shouldsave = false
	return nil
}

// clear this cookie, by set Expires to now
func (s *session) Clear() error {
	s.shouldsave = false
	return nil
}

/*
// return should save
func (s *session) ShouldSave() bool {
	return s.shouldsave
}

// set should save
// if shouldsave is true, the handler should add set-cookie to request header to
// set this cookie
func (s *session) SetShouldSave(ss bool) {
	s.shouldsave = ss
}

// return shouldset
func (s *session) ShouldSet() bool {
	return s.shouldset
}

// if shouldset is true, when the request is finished, the session data is set
// back to store
func (s *session) SetShouldSet(ss bool) {
	s.shouldset = ss
}
*/

func (s *session) AddFlash(value interface{}) {
	var flashes []interface{}

	if v, ok := s.data[flashesKey]; ok {
		flashes = v.([]interface{})
	}
	s.data[flashesKey] = append(flashes, value)
}

func (s *session) Flashes() []interface{} {
	var flashes []interface{}

	if v, ok := s.data[flashesKey]; ok {
		// Drop the flashes and return it.
		delete(s.data, flashesKey)
		flashes = v.([]interface{})
	}

	return flashes
}
