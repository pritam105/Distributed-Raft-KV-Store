package raft

import (
	"fmt"
	"sync/atomic"
)

type NodeState int32

const (
	Follower NodeState = iota
	Candidate
	Leader
)

func (s NodeState) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return fmt.Sprintf("Unknown(%d)", int(s))
	}
}

type atomicState struct {
	v int32
}

func (a *atomicState) load() NodeState {
	return NodeState(atomic.LoadInt32(&a.v))
}

func (a *atomicState) store(s NodeState) {
	atomic.StoreInt32(&a.v, int32(s))
}

func (a *atomicState) cas(old, new NodeState) bool {
	return atomic.CompareAndSwapInt32(&a.v, int32(old), int32(new))
}
