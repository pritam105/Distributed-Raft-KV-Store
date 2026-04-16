package raft

import (
	"fmt"
	"sync"
	"time"
)

type Cluster struct {
	mu       sync.RWMutex
	nodes    map[string]*Node
	isolated map[string]bool
	latency  time.Duration
}

func NewCluster(n int, latency time.Duration) (*Cluster, []*Node) {
	c := &Cluster{
		nodes:    make(map[string]*Node),
		isolated: make(map[string]bool),
		latency:  latency,
	}

	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("node%c", rune('A'+i))
	}

	nodes := make([]*Node, n)
	for i, id := range ids {
		peers := make([]string, 0, n-1)
		for _, pid := range ids {
			if pid != id {
				peers = append(peers, pid)
			}
		}
		nodes[i] = NewNode(id, peers, c.transportFor(id), nil)
		c.nodes[id] = nodes[i]
	}
	return c, nodes
}

func (c *Cluster) Isolate(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isolated[id] = true
	fmt.Printf("[cluster] ISOLATED %s\n", id)
}

func (c *Cluster) Restore(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.isolated, id)
	fmt.Printf("[cluster] RESTORED %s\n", id)
}

func (c *Cluster) AddNode(nd *Node) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nodes[nd.id] = nd
}

func (c *Cluster) transportFor(senderID string) Transport {
	return &clusterTransport{sender: senderID, cluster: c}
}

type clusterTransport struct {
	sender  string
	cluster *Cluster
}

func (t *clusterTransport) SendRequestVote(peerID string, args RequestVoteArgs) (RequestVoteReply, error) {
	peer, err := t.resolve(peerID)
	if err != nil {
		return RequestVoteReply{}, err
	}
	return peer.HandleRequestVote(args), nil
}

func (t *clusterTransport) SendAppendEntries(peerID string, args AppendEntriesArgs) (AppendEntriesReply, error) {
	peer, err := t.resolve(peerID)
	if err != nil {
		return AppendEntriesReply{}, err
	}
	return peer.HandleAppendEntries(args), nil
}

func (t *clusterTransport) resolve(peerID string) (*Node, error) {
	c := t.cluster
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.isolated[t.sender] || c.isolated[peerID] {
		return nil, ErrNotReachable{PeerID: peerID}
	}

	peer, ok := c.nodes[peerID]
	if !ok {
		return nil, ErrNotReachable{PeerID: peerID}
	}

	if c.latency > 0 {
		time.Sleep(c.latency)
	}

	return peer, nil
}
