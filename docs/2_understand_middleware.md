Hope You read the previous one. If not, go back and read it first.


> "that method belongs to RateLimiter"

✅ Correct.

```go
type RateLimiter struct {
    next http.Handler
}

func (rl *RateLimiter) ServeHTTP(
    w http.ResponseWriter,
    r *http.Request,
) {
    ...
}
```

The receiver is

```go
(rl *RateLimiter)
```

which means the method belongs to `*RateLimiter`.

---

Now ask yourself:

What does the `http.Handler` interface require?

```go
type Handler interface {
    ServeHTTP(http.ResponseWriter, *http.Request)
}
```

Does `*RateLimiter` have a method with exactly that signature?

**Yes.**

Therefore,

```
*RateLimiter
      │
Has ServeHTTP()
      │
      ▼
Implements http.Handler
```

Notice something beautiful.

There is **no inheritance**.

There is **no `implements` keyword**.

There is **no registration**.

The compiler simply checks:

> "Does this type have a `ServeHTTP(http.ResponseWriter, *http.Request)` method?"

If yes,

> "Then it satisfies `http.Handler`."

---

# This is why middleware works

Now imagine this:

```go
type RateLimiter struct {
    next http.Handler
}
```

What is `next`?

Not a function.

Not a `ServeMux`.

Not a `Hello` handler.

Just...

```go
http.Handler
```

That means `next` could be:

* a `ServeMux`
* a `HandlerFunc`
* another middleware
* a Gin adapter
* a custom handler
* literally **anything** that implements `ServeHTTP`.

This is the power of programming to an interface.

---

# Now imagine the request flow

Suppose your middleware does:

```go
func (rl *RateLimiter) ServeHTTP(w http.ResponseWriter, r *http.Request) {

    if allowed {
        rl.next.ServeHTTP(w, r)
        return
    }

    http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
}
```

Read it in English:

```
Request arrives

↓

Check rate limit

↓

Allowed?

     │
  Yes │ No
     │
     ▼
Call next handler

or

Return 429
```

That single line

```go
rl.next.ServeHTTP(w, r)
```

is what makes middleware a **chain**.

---

# Here's the biggest realization

Remember how the server worked?

```
Server

↓

handler.ServeHTTP()
```

The server doesn't know whether `handler` is:

* a `ServeMux`
* a `RateLimiter`
* a logger
* an authentication middleware

It just calls:

```go
handler.ServeHTTP(...)
```

Now imagine the handler is actually your middleware.

The server calls:

```
RateLimiter.ServeHTTP()

↓

RateLimiter calls

next.ServeHTTP()

↓

ServeMux.ServeHTTP()

↓

HandlerFunc.ServeHTTP()

↓

Hello()
```

See what happened?

The server thinks it's talking to **one handler**.

But internally, that handler delegated to another handler, which delegated to another, and so on.

---

# The complete chain

```
http.Server
      │
      ▼
RateLimiter Middleware
      │
      ▼
Logging Middleware
      │
      ▼
ServeMux
      │
      ▼
HandlerFunc
      │
      ▼
Hello()
```

Every single box implements the same interface:

```go
type Handler interface {
    ServeHTTP(http.ResponseWriter, *http.Request)
}
```

That's why they are interchangeable.

---

[Read next here](./write_these_before_middleware.md)