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
	// Get returns the session value associated to the given key.
	Get(key interface{}) interface{}
	// Create a new session ID with sessiondata
	Create()
	// Set sets the session value associated to the given key.
	Set(key interface{}, val interface{})
	// Save is to the client, usualy browsers
	Save() error
	// return should save
	ShouldSave() bool
	// set should save
	SetShouldSave(ss bool)
	// AddFlash adds a flash message to the session.
	// A single variadic argument is accepted, and it is optional: it defines the flash key.
	// If not defined "_flash" is used by default.
	AddFlash(value interface{}, vars ...string)
	// Flashes returns a slice of flash messages from the session.
	// A single variadic argument is accepted, and it is optional: it defines the flash key.
	// If not defined "_flash" is used by default.
	Flashes(vars ...string) []interface{}
}

var (
	store       Store
	sessionname string = "sid"
	secretKey   []byte
)

// Sessions is a Middleware that maps a session.Session service into the Martini handler chain.
// Sessions can use a number of storage solutions with the given store options.
// store: memory, redis, memcache, etc
// options: usally json string to open store
func Sessions(name string, storetype string, dsn string, secret string) martini.Handler {
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

	return func(res http.ResponseWriter, r *http.Request, c martini.Context, l *log.Logger) {
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

		if s.ShouldSave() {
			err := s.Save()
			if err != nil {
				l.Printf(errorFormat, err)
			}
		}
	}
}

type sessiondata map[interface{}]interface{}

type session struct {
	key        string
	cookie     *http.Cookie
	data       *sessiondata
	shouldsave bool
}

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

	return (*s.data)[key]
}

// Create a new session ID with sessiondata
func (s *session) Create(l *log.Logger) {
	if s.data != nil {
		if l != nil {
			l.Println(warnFormat, "Overwrite exist session "+s.key)
		}
	}

	uuid, err := simpleuuid.NewTime(time.Now())
	if err != nil {
		panic(err) // I don't think this can actually happen.
	}

	s.key = hex.EncodeToString(uuid[0:16])
	data := (make(sessiondata))
	s.data = &data
}

func (s *session) Set(key interface{}, val interface{}) {
	if s.data == nil {
		s.Create(nil)
	}
}

// Save is to the client, usualy browsers
func (s *session) Save() error {
	return nil
}

// return should save
func (s *session) ShouldSave() bool {
	return s.shouldsave
}

// set should save
func (s *session) SetShouldSave(ss bool) {
	s.shouldsave = ss
}

const flashesKey = "_flash"

func (s *session) AddFlash(value interface{}, vars ...string) {
	key := flashesKey
	if len(vars) > 0 {
		key = vars[0]
	}
	var flashes []interface{}
	if v, ok := (*s.data)[key]; ok {
		flashes = v.([]interface{})
	}
	(*s.data)[key] = append(flashes, value)
}

func (s *session) Flashes(vars ...string) []interface{} {
	var flashes []interface{}
	key := flashesKey
	if len(vars) > 0 {
		key = vars[0]
	}
	if v, ok := (*s.data)[key]; ok {
		// Drop the flashes and return it.
		delete(*s.data, key)
		flashes = v.([]interface{})
	}
	return flashes
}
