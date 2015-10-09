package webrpc

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"reflect"

	"github.com/gorilla/websocket"
)

// Client holds an RPC client connection.
type Client struct {
	EventHandler
	ws    *websocket.Conn
	sendq chan Message
	recvq chan Message
	replm map[int]chan<- Message
	ack   int
}

// Dial connects to an RPC server.
func Dial(uri string) (*Client, error) {
	// Parse the URL.
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	// Actual dial.
	o := http.Header{"Origin": {uri}}
	c, err := net.Dial("tcp", u.Host)
	if err != nil {
		return nil, fmt.Errorf("Dial: %v", err)
	}

	// WebSocket handshake.
	ws, _, err := websocket.NewClient(c, u, o, 1024, 1024)
	if err != nil {
		return nil, err
	}

	// Create the client object.
	client := Client{
		ws:    ws,
		sendq: make(chan Message, 256),
		recvq: make(chan Message, 256),
		replm: make(map[int]chan<- Message),
	}

	client.EventHandler = EventHandler{
		handlers: map[string]reflect.Value{},
		sender:   &client,
	}

	// Start the message loops.
	go client.readLoop()
	go client.writeLoop()

	// Done.
	return &client, nil
}

// Close shuts down a connection.
func (c *Client) Close() error {
	return c.ws.Close()
}

// Emit calls a Socket.io function.
func (c *Client) Emit(name string, args ...interface{}) error {
	var callback reflect.Value
	var cbchan chan Message

	if len(args) > 0 {
		last := reflect.ValueOf(args[len(args)-1])
		if last.Kind() == reflect.Func {
			callback = last
			args = args[:len(args)-1]
		}
	}

	msg, err := NewEvent(name, args...)
	if err != nil {
		return err
	}

	if callback.IsValid() {
		c.ack++
		msg.Ack = c.ack
		cbchan = make(chan Message)
		c.replm[c.ack] = cbchan
	}

	c.sendq <- msg

	if msg.Ack > 0 {
		reply := <-cbchan
		delete(c.replm, c.ack)
		close(cbchan)

		_, err = callWith(callback, reply.Data, false)
		if err != nil {
			return err
		}
	}

	return nil
}

// NextMessage gets the next message.
func (c *Client) NextMessage() Message {
	return <-c.recvq
}

// Dispatch runs a dispatch loop.
func (c *Client) Dispatch() {
	for {
		select {
		case message, ok := <-c.recvq:
			if !ok {
				return
			}

			c.dispatch(message)
		}
	}
}

func (c *Client) writeLoop() {
	defer func() {
		c.ws.Close()
	}()

	for {
		select {
		case message, ok := <-c.sendq:
			if !ok {
				return
			}
			if err := c.ws.WriteJSON(message); err != nil {
				return
			}
		}
	}
}

func (c *Client) readLoop() {
	defer func() {
		close(c.recvq)
		c.ws.Close()
	}()

	for {
		message := Message{}
		err := c.ws.ReadJSON(&message)
		if err != nil {
			return
		}

		switch message.Type {
		case Init:
			c.sendq <- newInit()
		case Ping:
			c.sendq <- newPong()
		case Reply:
			cbchan, ok := c.replm[message.Ack]
			if !ok {
				log.Printf("invalid ack")
				return
			}
			cbchan <- message
		case Event:
			c.recvq <- message
		default:
			log.Printf("invalid message")
			return
		}
	}
}

func (c *Client) send(msg Message) {
	c.sendq <- msg
}
