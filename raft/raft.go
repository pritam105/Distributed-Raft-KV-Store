package raft

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

const (
	heartbeatInterval  = 100 * time.Millisecond
	electionTimeoutMin = 300 * time.Millisecond
	electionTimeoutMax = 600 * time.Millisecond
)

func newElectionTimeout() time.Duration {
	spread := int64(electionTimeoutMax - electionTimeoutMin)
	return electionTimeoutMin + time.Duration(rand.Int63n(spread))
}

type NodeMetrics struct {
	mu               sync.Mutex
	ElectionsStarted int
	VotesGranted     int
	HeartbeatsSent   int
	HeartbeatsRecvd  int
	CurrentLeaderID  string
}

func (m *NodeMetrics) Snapshot() NodeMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()
	return NodeMetrics{
		ElectionsStarted: m.ElectionsStarted,
		VotesGranted:     m.VotesGranted,
		HeartbeatsSent:   m.HeartbeatsSent,
		HeartbeatsRecvd:  m.HeartbeatsRecvd,
		CurrentLeaderID:  m.CurrentLeaderID,
	}
}

type Node struct {
	id    string
	peers []string

	mu          sync.Mutex
	currentTerm uint64
	votedFor    string
	log         *logStore

	state atomicState

	transport Transport
	Metrics   *NodeMetrics

	resetTimerCh chan struct{}
	stepDownCh   chan uint64
	stopCh       chan struct{}
}

func NewNode(id string, peers []string, t Transport) *Node {
	n := &Node{
		id:           id,
		peers:        peers,
		log:          newLogStore(),
		transport:    t,
		Metrics:      &NodeMetrics{},
		resetTimerCh: make(chan struct{}, 8),
		stepDownCh:   make(chan uint64, 4),
		stopCh:       make(chan struct{}),
	}
	n.state.store(Follower)
	return n
}

func (n *Node) ID() string       { return n.id }
func (n *Node) State() NodeState { return n.state.load() }
func (n *Node) IsLeader() bool   { return n.state.load() == Leader }

func (n *Node) Term() uint64 {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.currentTerm
}

func (n *Node) LeaderID() string {
	return n.Metrics.Snapshot().CurrentLeaderID
}

func (n *Node) Start() {
	go n.run()
}

func (n *Node) Stop() {
	select {
	case <-n.stopCh:
	default:
		close(n.stopCh)
	}
}

func (n *Node) run() {
	for {
		select {
		case <-n.stopCh:
			return
		default:
		}
		switch n.state.load() {
		case Follower:
			n.loopFollower()
		case Candidate:
			n.loopCandidate()
		case Leader:
			n.loopLeader()
		}
	}
}

func (n *Node) becomeFollower(term uint64) {
	if term > n.currentTerm {
		log.Printf("[%s] term %d -> %d becoming Follower",
			n.id, n.currentTerm, term)
		n.currentTerm = term
		n.votedFor = ""
	}
	n.state.store(Follower)
}

func (n *Node) signalReset() {
	select {
	case n.resetTimerCh <- struct{}{}:
	default:
	}
}

func (n *Node) signalStepDown(term uint64) {
	select {
	case n.stepDownCh <- term:
	default:
	}
}
