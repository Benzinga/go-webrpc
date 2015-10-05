package webrpc

import (
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestServer() *httptest.Server {
	i := 0
	mutex := sync.RWMutex{}
	globalnicks := map[string]struct{}{}
	rpcserv := NewServer()

	rpcserv.OnConnect(func(c *Conn) {
		chans := map[string]struct{}{}
		nick := ""

		// Set user nick.
		setNick := func(new string) bool {
			// Lock mutex to block race conditions.
			mutex.Lock()
			defer mutex.Unlock()

			// Fail if name in use.
			if _, ok := globalnicks[new]; ok {
				return false
			}

			// Broadcast change.
			for ch := range chans {
				err := c.Broadcast(ch, "nick", nick, new)
				if err != nil {
					panic(err)
				}
			}

			// Update global nick map.
			delete(globalnicks, nick)
			globalnicks[new] = struct{}{}
			nick = new

			return true
		}

		c.On("nick", setNick)

		c.On("join", func(ch string) bool {
			c.Join(ch)
			err := c.Broadcast(ch, "join", ch, nick)

			if err != nil {
				return false
			}

			chans[ch] = struct{}{}
			return true
		})

		c.On("part", func(ch string) bool {
			err := c.Broadcast(ch, "part", ch, nick)
			c.Leave(ch)

			if err != nil {
				return false
			}

			delete(chans, ch)
			return true
		})

		c.On("message", func(ch string, msg string) bool {
			return c.Broadcast(ch, "message", ch, nick, msg) == nil
		})

		c.On("action", func(ch string, msg string) bool {
			return c.Broadcast(ch, "action", ch, nick, msg) == nil
		})

		c.On("quit", func() {
			c.Close()
		})

		c.OnClose(func() {
			return
		})

		setNick("Guest" + strconv.Itoa(100000+i))
		c.Emit("nick", "", nick)
		i++
	})

	return httptest.NewServer(rpcserv)
}

func otherClient(ts *httptest.Server) {
	cl, err := Dial(ts.URL)
	if err != nil {
		panic(err)
	}
	defer cl.Close()

	cl.Emit("join", "#general")
	cl.Emit("nick", "Test2")
	cl.Emit("message", "#general", "Hello!")
	cl.Emit("message", "#general", "Anyone around?")
	cl.Emit("part", "#general")

	cl.Emit("join", "#riot")
	cl.Emit("message", "#riot", "Hello?")
	cl.Emit("part", "#riot")

	cl.Emit("quit")
	cl.Dispatch()
}

func TestServer(t *testing.T) {
	ts := makeTestServer()
	defer ts.Close()

	cl, err := Dial(ts.URL)
	require.Nil(t, err)
	defer cl.Close()

	nicks := [][2]string{
		{"", "Guest100000"},
		{"Guest100001", "Test2"},
	}

	messages := [][3]string{
		{"#general", "Test2", "Hello!"},
		{"#general", "Test2", "Anyone around?"},
	}

	cl.On("nick", func(old string, new string) {
		assert.Equal(t, nicks[0][0], old)
		assert.Equal(t, nicks[0][1], new)
		nicks = nicks[1:]
	})

	cl.On("message", func(chname string, name string, message string) {
		assert.Equal(t, messages[0][0], chname)
		assert.Equal(t, messages[0][1], name)
		assert.Equal(t, messages[0][2], message)
		messages = messages[1:]
	})

	cl.Emit("join", "#general")
	cl.Emit("nick", "Test1")
	otherClient(ts)
	cl.Emit("quit")

	cl.Dispatch()

	// Make sure all assertions were hit.
	assert.Empty(t, nicks)
	assert.Empty(t, messages)
}
