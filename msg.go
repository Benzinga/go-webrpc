package webrpc

import "encoding/json"

// MessageType represents the kind of low-level message being received.
type MessageType int

// This is an enumeration of possible event types.
const (
	Init  MessageType = 0
	Event MessageType = 1
	Reply MessageType = 2
	Ping  MessageType = 3
	Pong  MessageType = 4
)

// Message represents a raw RPC message.
type Message struct {
	Type MessageType       `json:"type"`
	Ack  int               `json:"ack,omitempty"`
	Name string            `json:"name,omitempty"`
	Data []json.RawMessage `json:"data,omitempty"`
}

// NewEvent creates a new event message.
func NewEvent(name string, args ...interface{}) (Message, error) {
	msg := Message{
		Type: Event,
		Name: name,
	}

	err := msg.SetData(args...)
	if err != nil {
		return Message{}, err
	}

	return msg, nil
}

// NewReply creates a new reply message.
func NewReply(ack int, name string, args ...interface{}) (Message, error) {
	msg := Message{
		Type: Reply,
		Ack:  ack,
		Name: name,
	}

	err := msg.SetData(args...)
	if err != nil {
		return Message{}, err
	}

	return msg, nil
}

func newPing() Message {
	return Message{Type: Ping}
}

func newPong() Message {
	return Message{Type: Pong}
}

func newInit() Message {
	return Message{Type: Init}
}

// SetData sets the data to the provided arguments, returning an error if the
// data could not be marshalled.
func (m *Message) SetData(args ...interface{}) error {
	var err error

	l := len(args)
	m.Data = make([]json.RawMessage, l)

	for i := 0; i < l; i++ {
		m.Data[i], err = json.Marshal(args[i])
		if err != nil {
			return err
		}
	}

	return nil
}
