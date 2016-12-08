package webrpc

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Server implements an RPC server.
type Server struct {
	chans     map[string]*channel
	onConnect func(c *Conn)
	upgrader  websocket.Upgrader
}

// Config specifies a configuration for the server.
type Config struct {
	ReadBufferSize, WriteBufferSize int
	EnableCompression               bool
}

// NewServer creates a new server instance.
func NewServer() *Server {
	return NewServerWithConfig(Config{
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		EnableCompression: true,
	})
}

// NewServerWithConfig creates a new server with config.
func NewServerWithConfig(c Config) *Server {
	return &Server{
		chans: map[string]*channel{},
		upgrader: websocket.Upgrader{
			ReadBufferSize:    c.ReadBufferSize,
			WriteBufferSize:   c.WriteBufferSize,
			EnableCompression: c.EnableCompression,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
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

	ws, err := s.upgrader.Upgrade(w, r, nil)
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
