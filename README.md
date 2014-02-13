msession
========

A session implement for martini(a golang web framework)

Usage:

First, you should add a param like this:

    func FooHandler(w http.ResponseWriter, req *http.Request, 
        sess session.Session) {
    }

Second, in the handler function, to use the session, you must
first init it, like this:

    sess.Init()

Then, you can use the session as following:

## 1. Get

    Get(key interface{}) interface{}

  for example:

    sess.Get("uid")

## 2. Create

    Create(age int, l *log.Logger)

  create a cookie, age is in second, l is optional loger, for example:

    sess.Create(3600, nil)

## 3. SetKey

    SetKey(key interface{}, val interface{})

## 4. DelKey

	DelKey(key interface{})
	
## 5. Refresh

    sess.Refresh(d time.Duration)

## 6. Clear

    sess.Clear()

## 7. Save

    sess.Save()

## 8. AddFlash

    sess.AddFlash(value interface{})

## 9. Flashes()

    sess.Frashes() []interface{}

