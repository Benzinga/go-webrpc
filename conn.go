package webrpc

import (
	"errors"
	"log"
	"net"
	"reflect"
	"time"

	"github.com/gorilla/websocket"
)

// Common handler errors.
var (
	ErrNotInChan = errors.New("not in channel")
)

const (
	writeWait   = 10 * time.Second
	pingTimeout = 60 * time.Second
	pingPeriod  = 20 * time.Second
	sendqLength = 4096
)

// Conn represents an RPC connection.
type Conn struct {
	EventHandler
	s       *Server
	ws      *websocket.Conn
	sendq   chan Message
	chans   map[string]*channel
	onError func(error)
	onClose func()
}

func newConn(s *Server, ws *websocket.Conn) *Conn {
	conn := Conn{
		s:     s,
		ws:    ws,
		sendq: make(chan Message, sendqLength),
		chans: map[string]*channel{},
	}

	conn.EventHandler = EventHandler{
		handlers: map[string]reflect.Value{},
		sender:   &conn,
	}

	return &conn
}

// Close closes the underlying connection.
func (c *Conn) Close() error {
	return c.ws.Close()
}

// Emit sends an event to the client.
func (c *Conn) Emit(name string, args ...interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(string); ok {
				err = errors.New(e)
			}
		}
	}()

	msg, err := NewEvent(name, args...)
	if err != nil {
		return err
	}

	c.send(msg)
	return nil
}

// Broadcast sends an event to a channel. This function fails if the user is not
// in the channel and returns ErrNotInChan. Note that this method doesn't send
// the event back to the user who sent it; for that, use Server.Broadcast
// instead.
func (c *Conn) Broadcast(chname, name string, args ...interface{}) error {
	ch, ok := c.chans[chname]
	if !ok {
		return ErrNotInChan
	}

	msg, err := NewEvent(name, args...)
	if err != nil {
		return err
	}

	ch.broadcast(msg, c)
	return nil
}

// Join adds the user to a channel.
func (c *Conn) Join(chname string) {
	c.joinChan(c.s.getChannel(chname))
}

// Leave removes the user from a channel.
func (c *Conn) Leave(chname string) {
	ch, ok := c.chans[chname]
	if !ok {
		return
	}
	c.leaveChan(ch)
}

// Addr returns the remote address of the underlying connection.
func (c *Conn) Addr() net.Addr {
	return c.ws.RemoteAddr()
}

// readLoop is the read loop; note: this is where ALL callbacks run.
func (c *Conn) readLoop() {
	defer func() {
		if c.onClose != nil {
			c.onClose()
		}
		c.ws.Close()
	}()

	c.ws.SetReadDeadline(time.Now().Add(pingTimeout))
	for {
		msg := Message{}
		err := c.ws.ReadJSON(&msg)

		if err != nil {
			if c.onError != nil {
				c.onError(err)
			}
			return
		}

		if msg.Type == Pong {
			c.ws.SetReadDeadline(time.Now().Add(pingTimeout))
			continue
		}

		if msg.Type == Init {
			continue
		}

		// Dispatch message to handler.
		err = c.dispatch(msg)
		if err != nil {
			if c.onError != nil {
				c.onError(err)
			}
			return
		}
	}
}

// OnError sets the error handler for a socket.
func (c *Conn) OnError(handler func(error)) {
	c.onError = handler
}

// OnClose sets the close handler for a socket.
func (c *Conn) OnClose(handler func()) {
	c.onClose = handler
}

func (c *Conn) write(mt int, payload Message) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteJSON(payload)
}

func (c *Conn) writeRaw(mt int) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(websocket.TextMessage, []byte{})
}

func (c *Conn) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		// Make sure sends do not deadlock.
		go func() {
			for range c.sendq {
			}
		}()

		// Leave all channels.
		c.leaveChans()
		ticker.Stop()
		c.ws.Close()

		// Also ends consumption loop above.
		close(c.sendq)
	}()

	c.ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(pingTimeout))

	if err := c.write(websocket.TextMessage, newInit()); err != nil {
		return
	}

	for {
		select {
		case message, ok := <-c.sendq:
			if !ok {
				c.writeRaw(websocket.CloseMessage)
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.TextMessage, newPing()); err != nil {
				return
			}
		}
	}
}

func (c *Conn) send(msg Message) {
	select {
	case c.sendq <- msg:
		return
	default:
		log.Println("sendq exceeded for " + c.Addr().String())
		c.ws.Close()
	}
}
