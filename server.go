package rpc

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Server implements an RPC server.
type Server struct {
	chans     map[string]*channel
	onConnect func(c *Conn)
}

// NewServer creates a new server instance.
func NewServer() *Server {
	return &Server{
		chans: map[string]*channel{},
	}
}

func (s *Server) getChannel(name string) *channel {
	ch, ok := s.chans[name]
	if !ok {
		ch = newChannel(name)
		s.chans[name] = ch
	}
	return ch
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	c := newConn(s, ws)
	go c.writeLoop()
	if s.onConnect != nil {
		s.onConnect(c)
	}
	c.readLoop()
}

// OnConnect sets the connection handler for this server.
func (s *Server) OnConnect(handler func(c *Conn)) {
	s.onConnect = handler
}

// Broadcast sends a message on a given channel.
func (s *Server) Broadcast(chname, name string, args ...interface{}) error {
	msg, err := NewEvent(name, args...)
	if err != nil {
		return err
	}

	s.getChannel(chname).broadcast(msg, nil)
	return nil
}
