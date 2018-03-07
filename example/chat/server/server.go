package main

import (
	"log"
	"net/http"
	"time"

	"github.com/Benzinga/go-webrpc"
)

func main() {
	log.Println("Server starting.")
	server := webrpc.NewServer()

	server.OnConnect(func(c *webrpc.Conn) {
		user := c.Addr().String()
		log.Println(user, "connected")

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

		c.On("msg", msg)
		join("#welcome")
		c.OnClose(func() {
			part("#welcome")
		})
	})

	log.Fatalln(http.ListenAndServe(":4321", server))
}
