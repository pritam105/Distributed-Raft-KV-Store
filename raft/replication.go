package raft

import (
	"errors"
	"fmt"
	"log"
	"time"
)

var ErrNotLeader = errors.New("not the leader")
var ErrReplicationTimeout = errors.New("replication timed out: no majority")

// Replicate appends a command to the Raft log and waits until a majority of
// nodes have confirmed it. Returns nil when the entry is committed.
// Must only be called on the leader.
func (n *Node) Replicate(command []byte) error {
	n.mu.Lock()
	if !n.IsLeader() {
		leaderID := n.Metrics.Snapshot().CurrentLeaderID
		n.mu.Unlock()
		return fmt.Errorf("%w: try %s", ErrNotLeader, leaderID)
	}

	lastIdx, _ := n.log.lastIndexAndTerm()
	entry := LogEntry{
		Term:    n.currentTerm,
		Index:   lastIdx + 1,
		Command: command,
	}
	n.log.appendEntry(entry)
	n.matchIndex[n.id] = entry.Index
	term := n.currentTerm
	commitIndex := n.commitIndex
	n.mu.Unlock()

	majority := (len(n.peers)+1)/2 + 1
	ackCh := make(chan bool, len(n.peers))

	for _, peer := range n.peers {
		go func(p string) {
			ackCh <- n.replicateToPeer(p, entry, term, commitIndex)
		}(peer)
	}

	// self counts as one ack
	acks := 1
	timeout := time.NewTimer(2 * time.Second)
	defer timeout.Stop()

	for range n.peers {
		select {
		case ok := <-ackCh:
			if ok {
				acks++
				if acks >= majority {
					n.mu.Lock()
					if entry.Index > n.commitIndex {
						n.commitIndex = entry.Index
					}
					n.mu.Unlock()
					n.applyCommitted()
					return nil
				}
			}
		case <-timeout.C:
			return ErrReplicationTimeout
		case <-n.stopCh:
			return ErrNotLeader
		}
	}

	return ErrReplicationTimeout
}

// replicateToPeer sends log entries to one peer, retrying with a lower nextIndex
// when the follower rejects due to log inconsistency (gap from concurrent writes).
func (n *Node) replicateToPeer(peer string, entry LogEntry, term uint64, leaderCommit uint64) bool {
	for {
		n.mu.Lock()
		if n.currentTerm != term || !n.IsLeader() {
			n.mu.Unlock()
			return false
		}
		nextIdx := n.nextIndex[peer]

		// Send all entries from nextIdx up to (and including) the target entry.
		var entries []LogEntry
		for idx := nextIdx; idx <= entry.Index; idx++ {
			if e, ok := n.log.entryAt(idx); ok {
				entries = append(entries, e)
			}
		}

		prevLogIndex := nextIdx - 1
		prevLogTerm := uint64(0)
		if prevLogIndex > 0 {
			if prev, ok := n.log.entryAt(prevLogIndex); ok {
				prevLogTerm = prev.Term
			}
		}
		n.mu.Unlock()

		args := AppendEntriesArgs{
			Term:         term,
			LeaderID:     n.id,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			Entries:      entries,
			LeaderCommit: leaderCommit,
		}

		reply, err := n.transport.SendAppendEntries(peer, args)
		if err != nil {
			return false
		}

		if reply.Term > term {
			n.signalStepDown(reply.Term)
			return false
		}

		if reply.Success {
			n.mu.Lock()
			if entry.Index > n.matchIndex[peer] {
				n.matchIndex[peer] = entry.Index
			}
			n.nextIndex[peer] = entry.Index + 1
			n.mu.Unlock()
			return true
		}

		// Follower rejected — decrement nextIndex and retry from earlier in the log.
		n.mu.Lock()
		if n.nextIndex[peer] > 1 {
			n.nextIndex[peer]--
		} else {
			n.mu.Unlock()
			return false
		}
		n.mu.Unlock()
	}
}

// sendHeartbeatToPeer sends a heartbeat (or outstanding log entries) to one peer.
// Called by the leader's heartbeat loop — this is also how straggler followers catch up.
func (n *Node) sendHeartbeatToPeer(peer string) {
	n.mu.Lock()
	term := n.currentTerm
	nextIdx := n.nextIndex[peer]
	lastIdx, _ := n.log.lastIndexAndTerm()
	commitIndex := n.commitIndex

	var entries []LogEntry
	var prevLogIndex, prevLogTerm uint64

	if nextIdx <= lastIdx {
		entries = n.log.entriesFrom(nextIdx)
		prevLogIndex = nextIdx - 1
		if prevLogIndex > 0 {
			if prev, ok := n.log.entryAt(prevLogIndex); ok {
				prevLogTerm = prev.Term
			}
		}
	}
	n.mu.Unlock()

	args := AppendEntriesArgs{
		Term:         term,
		LeaderID:     n.id,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      entries,
		LeaderCommit: commitIndex,
	}

	reply, err := n.transport.SendAppendEntries(peer, args)
	if err != nil {
		return
	}

	n.Metrics.mu.Lock()
	n.Metrics.HeartbeatsSent++
	n.Metrics.mu.Unlock()

	if reply.Term > term {
		n.signalStepDown(reply.Term)
		return
	}

	if len(entries) == 0 {
		return
	}

	if reply.Success {
		n.mu.Lock()
		newMatch := prevLogIndex + uint64(len(entries))
		if newMatch > n.matchIndex[peer] {
			n.matchIndex[peer] = newMatch
			n.nextIndex[peer] = newMatch + 1
		}
		n.advanceCommitIndex()
		n.mu.Unlock()
		n.applyCommitted()
	} else {
		n.mu.Lock()
		if n.nextIndex[peer] > 1 {
			n.nextIndex[peer]--
		}
		n.mu.Unlock()
	}
}

// advanceCommitIndex checks if any new entries can be committed based on
// matchIndex across the cluster. Must be called with n.mu held.
func (n *Node) advanceCommitIndex() {
	lastIdx, _ := n.log.lastIndexAndTerm()
	majority := (len(n.peers)+1)/2 + 1

	for idx := n.commitIndex + 1; idx <= lastIdx; idx++ {
		entry, ok := n.log.entryAt(idx)
		if !ok || entry.Term != n.currentTerm {
			continue
		}
		count := 1 // self
		for _, peer := range n.peers {
			if n.matchIndex[peer] >= idx {
				count++
			}
		}
		if count >= majority {
			n.commitIndex = idx
			log.Printf("[%s] committed index=%d", n.id, idx)
		}
	}
}

// applyCommitted pushes all entries between lastApplied and commitIndex
// onto applyCh so the KV apply loop can write them to the store.
func (n *Node) applyCommitted() {
	n.mu.Lock()
	var toApply []LogEntry
	for n.lastApplied < n.commitIndex {
		n.lastApplied++
		if entry, ok := n.log.entryAt(n.lastApplied); ok {
			toApply = append(toApply, entry)
		}
	}
	n.mu.Unlock()

	if n.applyCh == nil {
		return
	}

	for _, entry := range toApply {
		select {
		case n.applyCh <- entry.Command:
		case <-n.stopCh:
			return
		}
	}
}
