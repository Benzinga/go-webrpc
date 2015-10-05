package rpc

type channel struct {
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
	for peer := range c.peers {
		if peer == from {
			continue
		}

		peer.sendq <- msg
	}
}

func (c *Conn) joinChan(ch *channel) {
	c.chans[ch.name] = ch
	ch.peers[c] = struct{}{}
}

func (c *Conn) leaveChan(ch *channel) {
	delete(ch.peers, c)
	delete(c.chans, ch.name)
}

func (c *Conn) leaveChans() {
	for _, ch := range c.chans {
		delete(ch.peers, c)
	}
	c.chans = map[string]*channel{}
}
