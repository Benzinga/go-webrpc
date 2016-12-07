package webrpc

import "sync"

type channel struct {
	mutex sync.RWMutex
	name  string
	peers map[*Conn]struct{}
}

func newChannel(name string) *channel {
	return &channel{
		name:  name,
		peers: map[*Conn]struct{}{},
	}
}

func (c *channel) broadcast(msg Message, from *Conn) {
	c.mutex.RLock()
	for peer := range c.peers {
		if peer == from {
			continue
		}

		peer.send(msg)
	}
	c.mutex.RUnlock()
}

func (c *Conn) joinChan(ch *channel) {
	c.chans[ch.name] = ch

	ch.mutex.Lock()
	ch.peers[c] = struct{}{}
	ch.mutex.Unlock()
}

func (c *Conn) leaveChan(ch *channel) {
	ch.mutex.Lock()
	delete(ch.peers, c)
	ch.mutex.Unlock()

	delete(c.chans, ch.name)
}

func (c *Conn) leaveChans() {
	for _, ch := range c.chans {
		ch.mutex.Lock()
		delete(ch.peers, c)
		ch.mutex.Unlock()
	}
	c.chans = map[string]*channel{}
}
