[![godoc.org](http://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](https://godoc.org/github.com/Benzinga/go-webrpc) [![Build Status](https://travis-ci.org/Benzinga/go-webrpc.svg)](https://travis-ci.org/Benzinga/go-webrpc) [![codecov.io](http://codecov.io/github/Benzinga/go-webrpc/coverage.svg?branch=master)](http://codecov.io/github/Benzinga/go-webrpc?branch=master)

# Go-WebRPC
Go-WebRPC is an RPC-style communication library providing a thin abstraction over WebSockets.

# Getting Started
WebRPC is a library. To make use of it, you need to write software that imports it.

To make it easier to get started, a full demo is included that can be used with Docker Compose.

## Prerequisites
Go-WebRPC is built in the Go programming language. If you are new to Go, you will need to [install Go](https://golang.org/dl/).

You may want Docker in order to easily test the demo app, though it is not required to use `go-webrpc` or the demo app.

## Acquiring
Next, you'll want to `go get` go-webrpc, like so:

```sh
go get github.com/Benzinga/go-webrpc
```

If your `$GOPATH` is configured, and git is setup to know your credentials, in a few moments the command should complete with no output. The repository will exist under `$GOPATH/src/github.com/Benzinga/go-webrpc`. It cannot be moved from this location.

Hint: If you've never used Go before, your `$GOPATH` will be under the `go` folder of your user directory.

## Demo
In order to run the demo app, you'll need to change into the `example/chat` directory. Then, run `docker-compose up`.

```sh
cd example/chat
docker-compose up
```

Then, in a browser, visit http://localhost:1234. You should be able to chat with yourself.

### Server: Code Explanation
The server code is fairly simple, weighing around 40 lines of code.

First, the server is instantiated.
```go
    server := webrpc.NewServer()
```

Next, the `OnConnect` handler is set. This handler handles when a WebRPC client connects, and is similar to ServeHTTP.
```go
    server.OnConnect(func(c *webrpc.Conn) {
        ...
    })
```

In the OnConnect handler, we use the user's address as a sort of username.
```go
    server.OnConnect(func(c *webrpc.Conn) {
        user := c.Addr().String()

        ...
    })
```

We define a few helper functions. These deal with sending messages to the client.
```go
    server.OnConnect(func(c *webrpc.Conn) {
        ...

        join := func(ch string) {
            c.Join(ch)
            server.Broadcast(ch, "join", time.Now(), user, ch)
        }

        part := func(ch string) {
            server.Broadcast(ch, "part", time.Now(), user, ch)
            c.Leave(ch)
        }

        msg := func(ch string, msg string) {
            server.Broadcast(ch, "msg", time.Now(), user, msg)
        }

        ...
    })
```

To conclude our `OnConnect` handler, we do some things upon connecting as well as register a simple disconnect handler.
```go
    server.OnConnect(func(c *webrpc.Conn) {
        ...

        c.On("msg", msg)
        join("#welcome")
        c.OnClose(func() {
            part("#welcome")
        })
    })
```

Finally, we start an HTTP server with our WebRPC setup.
```go
    http.ListenAndServe(":4321", server)
```

### Client
The client is written in JavaScript and uses webrpc.js. This is out of scope for this README, but the code is in the `example/chat/client` folder of the repository.
