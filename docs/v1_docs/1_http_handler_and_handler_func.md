# Step 1: Forget HTTP for a moment

Let's start with something much simpler.

Imagine you have a function:

```go
func SayHello(name string) {
	fmt.Println("Hello", name)
}
```

Its type is:

```go
func(string)
```

Meaning:

> "A function that accepts a string and returns nothing."

Just like this:

```go
func Add(a int, b int) int
```

has type

```go
func(int, int) int
```

Every function in Go has a type.

---

# Step 2: Functions are values

This surprises many people coming from Python.

You can store a function in a variable.

```go
func SayHello(name string) {
	fmt.Println("Hello", name)
}

func main() {
	f := SayHello

	f("Nani")
}
```

Notice

```go
f := SayHello
```

Not

```go
f := SayHello()
```

We're storing the function itself.

---

You can even pass it around.

```go
func Execute(fn func(string)) {
	fn("Go")
}
```

Now

```go
Execute(SayHello)
```

works.

Again, the parameter type is

```go
func(string)
```

---

# Step 3: What is a Handler?

Now let's return to HTTP.

When a request comes in...

```
Browser

↓

GET /hello

↓

Server
```

The server needs **something** to process it.

What should that "something" be?

Go says:

> Just make it a function.

That function looks like this:

```go
func Hello(w http.ResponseWriter, r *http.Request) {

}
```

Its type is

```go
func(http.ResponseWriter, *http.Request)
```

Nothing magical.

It's just another function.

---

# Step 4: Why those two parameters?

Imagine someone visits

```
GET /hello
```

The server knows two important things.

## The request

Everything the client sent.

```
URL

Headers

Method

Body

Cookies
```

That's represented by

```go
*http.Request
```

---

## The response

The server also needs a way to send data back.

That's

```go
http.ResponseWriter
```

You don't return a response.

Instead, you write into it.

Like

```go
fmt.Fprintf(w, "Hello")
```

Think of it like writing into a network connection.

---

So the handler receives

```
Request

↓

Process

↓

Write Response
```

---

# Step 5: What does ServeMux store?

Earlier you wrote

```go
mux.HandleFunc("/hello", Hello)
```

Let's imagine how ServeMux works internally.

Very roughly:

```go
type ServeMux struct {
	routes map[string]???
}
```

Question:

What should the map store?

---

It could store

```
"/hello"

↓

Hello function
```

So conceptually

```go
routes["/hello"] = Hello
```

Now when

```
GET /hello
```

arrives,

ServeMux looks up

```
"/hello"
```

finds

```
Hello
```

and executes it.

Simple.

---

# Step 6: But there is a problem...

Suppose we want to store handlers in a map.

Can we write

```go
map[string]func(http.ResponseWriter, *http.Request)
```

?

Yes.

But Go chose a different design.

Instead it says

> "Anything capable of serving an HTTP request should satisfy one interface."

That interface is

```go
type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}
```

This is one of the most important interfaces in the standard library.

Read it carefully.

It says:

> Any type that has a method named `ServeHTTP` with this exact signature is an HTTP handler.

---

# Step 7: Interface satisfaction

Suppose we create

```go
type MyHandler struct{}
```

and

```go
func (h MyHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	fmt.Fprintln(w, "Hello")
}
```

Question:

Does `MyHandler` explicitly say

```go
implements Handler
```

?

No.

Go doesn't require that.

It automatically notices

```
MyHandler

↓

Has ServeHTTP()

↓

Therefore

↓

MyHandler satisfies Handler
```

This is called **implicit interface implementation**, one of Go's distinguishing features.

---

# Step 8: Then what is `HandleFunc`?

You wrote

```go
mux.HandleFunc("/hello", Hello)
```

But notice...

`Hello` is **not** a struct.

It doesn't have a

```go
ServeHTTP()
```

method.

So how can it satisfy `Handler`?

This is where the clever part begins.

The standard library defines

```go
type HandlerFunc func(http.ResponseWriter, *http.Request)
```

Read it carefully.

This is **not** a function.

It is a **named type whose underlying type is a function**.

---

Then the standard library adds a method:

```go
func (f HandlerFunc) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	f(w, r)
}
```

Let's slow down.

Inside

```go
ServeHTTP()
```

it simply calls the function itself.

```
ServeHTTP()

↓

Call stored function

↓

Done
```

---

# This is the magic

Suppose we have

```go
func Hello(
	w http.ResponseWriter,
	r *http.Request,
) {

}
```

Go can convert it into

```go
HandlerFunc(Hello)
```

Now it has a method

```
ServeHTTP()
```

which means

```
HandlerFunc

↓

implements Handler
```

Suddenly your ordinary function becomes an object satisfying the `Handler` interface.

---

# Visualizing it

```
Your function

Hello()

        │

        ▼

HandlerFunc(Hello)

        │

Has ServeHTTP()

        │

        ▼

Implements Handler
```

That's why `HandleFunc` exists.

It's simply a convenience.

Instead of making you write

```go
mux.Handle(
	"/hello",
	http.HandlerFunc(Hello),
)
```

you write

```go
mux.HandleFunc(
	"/hello",
	Hello,
)
```

Internally, the conversion happens for you.

---

# The actual `HandleFunc`

It's very close to this:

```go
func (mux *ServeMux) HandleFunc(
	pattern string,
	handler func(http.ResponseWriter, *http.Request),
) {
	mux.Handle(pattern, HandlerFunc(handler))
}
```

See what's happening?

1. Accept a plain function.
2. Convert it into `HandlerFunc`.
3. Store it as a `Handler`.

---

# Why does this matter?

Because middleware works with **`http.Handler`**, not with plain functions.

Middleware wraps one handler with another:

```
Incoming Request

↓

Logging Handler

↓

Rate Limiter Handler

↓

Authentication Handler

↓

Final Handler
```

Every box in that chain implements the same interface:

```go
type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}
```

That's what makes the pipeline composable.

---


### Mental model

```
You write

func Hello(w, r)

        │

        ▼

HandleFunc converts it into

http.HandlerFunc

        │

        ▼

HandlerFunc implements

http.Handler

        │

        ▼

ServeMux stores it

        │

        ▼

Server calls

handler.ServeHTTP(w, r)

        │

        ▼

HandlerFunc.ServeHTTP()

        │

        ▼

Hello(w, r)

```

> Next up Writing Middleware


But before going to it

One final question before middleware

Suppose instead of HandlerFunc, you create your own type:
```go
type RateLimiter struct {
    next http.Handler
}
```

If you add this method:
```go
func (rl *RateLimiter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // ...
}
```

Without using HandlerFunc at all, will *RateLimiter implement http.Handler?

[Check Here](./understand_middleware.md)