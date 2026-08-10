Hope you read the previous one. If not, go back and read it first.
---

# Step 1 — Our current server

Right now you have something like:

```go
mux := http.NewServeMux()

mux.HandleFunc("/hello", Hello)

server := &http.Server{
    Addr:    ":9090",
    Handler: mux,
}
```

Visually,

```text
http.Server
      │
      ▼
ServeMux
      │
      ▼
Hello()
```

---

# Step 2 — What does the server know?

This is extremely important.

The server does **not** know about:

* Hello
* Routes
* Middleware
* Rate limiting

It only knows one thing:

```go
handler.ServeHTTP(w, r)
```

where

```go
handler
```

is whatever you gave to

```go
server.Handler
```

---

## Question

What if...

Instead of giving the server

```go
Handler: mux
```

we gave it

```go
Handler: myMiddleware
```

Would the server care?

No.

Why?

Because if `myMiddleware` implements

```go
ServeHTTP(w, r)
```

then it **is** a `Handler`.

This is the first big idea.

---

# Step 3 — Build our own Handler

Let's write the smallest possible custom handler.

```go
type PrintHandler struct{}
```

Now make it satisfy `http.Handler`.

```go
func (p *PrintHandler) ServeHTTP(
    w http.ResponseWriter,
    r *http.Request,
) {
    fmt.Println("Inside PrintHandler")

    fmt.Fprintln(w, "Hello from PrintHandler")
}
```

Question:

Does this implement

```go
http.Handler
```

?

Yes.

So now we can do

```go
server := &http.Server{
    Addr: ":9090",
    Handler: &PrintHandler{},
}
```

No ServeMux.

No routes.

Nothing.

The server is happy.

---

# Step 4 — But we want routing

Our `PrintHandler` is boring.

We still want the mux.

So let's store it.

```go
type PrintHandler struct {
    next http.Handler
}
```

Notice something.

What type is `next`?

Not

```go
*ServeMux
```

Just

```go
http.Handler
```

This is intentional.

Because later it could be:

* ServeMux
* another middleware
* HandlerFunc
* anything

---

# Step 5 — The chain

Now write

```go
func (p *PrintHandler) ServeHTTP(
    w http.ResponseWriter,
    r *http.Request,
) {

    fmt.Println("Before next handler")

    p.next.ServeHTTP(w, r)

    fmt.Println("After next handler")
}
```

Stop.

Don't think about middleware.

Read this like English.

```text
Request arrives

↓

Print "Before"

↓

Call next handler

↓

Print "After"
```

That's all middleware is.

---

# Step 6 — Wiring

Suppose

```go
mux := http.NewServeMux()

mux.HandleFunc("/hello", Hello)
```

Now wrap it.

```go
printer := &PrintHandler{
    next: mux,
}
```

Then

```go
server := &http.Server{
    Addr: ":9090",
    Handler: printer,
}
```

Notice the chain.

```text
Server

↓

PrintHandler

↓

ServeMux

↓

Hello
```

---

# Step 7 — Request Flow

Suppose someone visits

```text
GET /hello
```

Exactly what happens?

```text
Server

↓

PrintHandler.ServeHTTP()

↓

Print

"Before next handler"

↓

ServeMux.ServeHTTP()

↓

Find "/hello"

↓

HandlerFunc.ServeHTTP()

↓

Hello()

↓

Return

↓

Print

"After next handler"
```

Beautiful.

No magic.

Just nested function calls.

---

# Step 8 — Let's make it more interesting

Modify

```go
ServeHTTP()
```

like this.

```go
func (p *PrintHandler) ServeHTTP(
    w http.ResponseWriter,
    r *http.Request,
) {
    start := time.Now()

    p.next.ServeHTTP(w, r)

    fmt.Println(time.Since(start))
}
```

What did we just build?

A timing middleware.

---

Or

```go
fmt.Println(r.Method)
```

Logging middleware.

---

Or

```go
fmt.Println(r.URL.Path)
```

Request logging middleware.

---

Or

```go
if !authenticated {
    http.Error(...)
    return
}

p.next.ServeHTTP(w, r)
```

Authentication middleware.

---

Or...

```go
if !rateLimiter.Allow(userID) {
    http.Error(...)
    return
}

p.next.ServeHTTP(w, r)
```

Rate limiter middleware.

See?

The only thing that changes is **what happens before calling `next`**.

The structure never changes.

---

# This is why middleware is powerful

Every middleware follows this pattern:

```text
Receive Request

↓

Do Something

↓

Continue?

Yes ──────────────► next.ServeHTTP()

No ───────────────► Return Response
```

The server doesn't know.

The mux doesn't know.

The handler doesn't know.

Only the middleware decides whether the request continues.

---

# 🧠 Small Exercise (Don't Skip)

Before we write the actual rate limiter middleware, I want you to implement this yourself.

Create:

```go
type PrintMiddleware struct {
    next http.Handler
}
```

Implement:

```go
ServeHTTP(...)
```

that prints

```text
========== Incoming Request ==========
GET /hello
======================================
```

before forwarding the request to `next`.

Then wire it like this:

```text
Server

↓

PrintMiddleware

↓

ServeMux

↓

Hello Handler
```

Run it.

Watch the logs.

The Snippets are in internal/logger/logger.go
and the middleware chaining is in cmd/server/routes.go

