package rpc

import (
	"encoding/json"
	"errors"
	"reflect"
)

// Common handler errors.
var (
	ErrBadArgLen   = errors.New("invalid number of arguments")
	ErrBadDispatch = errors.New("type mismatch")
	ErrNoData      = errors.New("no data")
	ErrNotFunc     = errors.New("not a function")
	ErrNilFunc     = errors.New("nil function")
)

// EventHandler provides a basic event handling system.
type EventHandler struct {
	handlers map[string]reflect.Value
	sender   MessageSender
}

// MessageSender represents classes that can send Messages.
type MessageSender interface {
	send(Message)
}

// callWith calls a function against a JSON array.
func callWith(fn reflect.Value, indata []json.RawMessage, ret bool) ([]json.RawMessage, error) {
	var err error

	t := fn.Type()
	l := len(indata)

	if l != t.NumIn() {
		return nil, ErrBadArgLen
	}

	args := make([]reflect.Value, l)

	for i := 0; i < l; i++ {
		arg := reflect.New(t.In(i)).Interface()
		err = json.Unmarshal(indata[i], arg)
		if err != nil {
			return nil, err
		}
		args[i] = reflect.ValueOf(arg).Elem()
	}

	rets := fn.Call(args)

	if ret {
		l = len(rets)
		outdata := make([]json.RawMessage, l)

		for i := 0; i < l; i++ {
			ret := rets[i].Interface()
			outdata[i], err = json.Marshal(ret)
			if err != nil {
				return nil, err
			}
		}

		return outdata, nil
	}

	return nil, nil
}

// dispatch calls an event handler for a message.
func (c *EventHandler) dispatch(event Message) error {
	handler, ok := c.handlers[event.Name]
	if !ok {
		return nil
	}

	ret, err := callWith(handler, event.Data, event.Ack != 0)

	if err != nil {
		return err
	}

	if ret != nil && c.sender != nil {
		reply := Message{
			Type: Reply,
			Ack:  event.Ack,
			Data: ret,
		}
		c.sender.send(reply)
	}

	return nil
}

// On connects a function to an event handler.
func (c *EventHandler) On(name string, fn interface{}) {
	if fn == nil {
		panic(ErrNilFunc)
	}

	value := reflect.ValueOf(fn)
	if value.Kind() != reflect.Func {
		panic(ErrNotFunc)
	}

	c.handlers[name] = value
}
