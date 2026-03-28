package raft

import "fmt"

type RequestVoteArgs struct {
	Term         uint64
	CandidateID  string
	LastLogIndex uint64
	LastLogTerm  uint64
}

type RequestVoteReply struct {
	Term        uint64
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term     uint64
	LeaderID string
}

type AppendEntriesReply struct {
	Term    uint64
	Success bool
}

type Transport interface {
	SendRequestVote(peerID string, args RequestVoteArgs) (RequestVoteReply, error)
	SendAppendEntries(peerID string, args AppendEntriesArgs) (AppendEntriesReply, error)
}

type ErrNotReachable struct {
	PeerID string
}

func (e ErrNotReachable) Error() string {
	return fmt.Sprintf("peer %s is not reachable", e.PeerID)
}
