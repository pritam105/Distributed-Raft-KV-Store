package raft

import (
	"log"
	"sync"
	"time"
)

func (n *Node) loopFollower() {
	timeout := newElectionTimeout()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	log.Printf("[%s] Follower  term=%d  timeout=%v", n.id, n.Term(), timeout)

	for {
		select {
		case <-n.stopCh:
			return

		case term := <-n.stepDownCh:
			n.mu.Lock()
			n.becomeFollower(term)
			n.mu.Unlock()
			return

		case <-n.resetTimerCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(newElectionTimeout())

		case <-timer.C:
			log.Printf("[%s] election timeout - becoming Candidate", n.id)
			n.state.store(Candidate)
			return
		}
	}
}
func (n *Node) loopCandidate() {
	n.mu.Lock()
	n.currentTerm++
	n.votedFor = n.id
	term := n.currentTerm
	lastIdx, lastTerm := n.log.lastIndexAndTerm()
	n.mu.Unlock()

	n.Metrics.mu.Lock()
	n.Metrics.ElectionsStarted++
	n.Metrics.mu.Unlock()

	log.Printf("[%s] Candidate  term=%d", n.id, term)

	electionTimer := time.NewTimer(newElectionTimeout())
	defer electionTimer.Stop()

	type result struct {
		granted bool
		term    uint64
	}
	resultCh := make(chan result, len(n.peers))

	for _, peer := range n.peers {
		go func(p string) {
			args := RequestVoteArgs{
				Term:         term,
				CandidateID:  n.id,
				LastLogIndex: lastIdx,
				LastLogTerm:  lastTerm,
			}
			reply, err := n.transport.SendRequestVote(p, args)
			if err != nil {
				resultCh <- result{granted: false}
				return
			}
			resultCh <- result{granted: reply.VoteGranted, term: reply.Term}
		}(peer)
	}

	votes := 1
	majority := (len(n.peers)+1)/2 + 1

	for range n.peers {
		select {
		case <-n.stopCh:
			return

		case higherTerm := <-n.stepDownCh:
			n.mu.Lock()
			n.becomeFollower(higherTerm)
			n.mu.Unlock()
			return

		case <-electionTimer.C:
			log.Printf("[%s] election timed out at term=%d retrying", n.id, term)
			return

		case res := <-resultCh:
			if res.term > term {
				n.mu.Lock()
				n.becomeFollower(res.term)
				n.mu.Unlock()
				return
			}
			if res.granted {
				votes++
				log.Printf("[%s] got vote %d/%d", n.id, votes, majority)
				if votes >= majority {
					n.becomeLeader(term)
					return
				}
			}
		}
	}
}

func (n *Node) becomeLeader(term uint64) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.currentTerm != term {
		return
	}
	if !n.state.cas(Candidate, Leader) {
		return
	}

	log.Printf("[%s] *** LEADER term=%d ***", n.id, term)

	n.Metrics.mu.Lock()
	n.Metrics.CurrentLeaderID = n.id
	n.Metrics.mu.Unlock()
}

func (n *Node) loopLeader() {
	log.Printf("[%s] Leader loop started term=%d", n.id, n.Term())

	n.sendHeartbeats()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-n.stopCh:
			return

		case higherTerm := <-n.stepDownCh:
			n.mu.Lock()
			n.becomeFollower(higherTerm)
			n.mu.Unlock()
			return

		case <-ticker.C:
			if n.state.load() != Leader {
				return
			}
			n.sendHeartbeats()
		}
	}
}

func (n *Node) sendHeartbeats() {
	n.mu.Lock()
	term := n.currentTerm
	leaderID := n.id
	n.mu.Unlock()

	var wg sync.WaitGroup
	for _, peer := range n.peers {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			args := AppendEntriesArgs{
				Term:     term,
				LeaderID: leaderID,
			}
			reply, err := n.transport.SendAppendEntries(p, args)
			if err != nil {
				return
			}
			n.Metrics.mu.Lock()
			n.Metrics.HeartbeatsSent++
			n.Metrics.mu.Unlock()

			if reply.Term > term {
				n.signalStepDown(reply.Term)
			}
		}(peer)
	}
	wg.Wait()
}

func (n *Node) HandleRequestVote(args RequestVoteArgs) RequestVoteReply {
	n.mu.Lock()
	defer n.mu.Unlock()

	reply := RequestVoteReply{Term: n.currentTerm}

	if args.Term < n.currentTerm {
		log.Printf("[%s] reject vote for %s stale term", n.id, args.CandidateID)
		return reply
	}

	if args.Term > n.currentTerm {
		n.becomeFollower(args.Term)
		reply.Term = n.currentTerm
	}

	alreadyVoted := n.votedFor != "" && n.votedFor != args.CandidateID
	myLastIdx, myLastTerm := n.log.lastIndexAndTerm()
	logOK := args.LastLogTerm > myLastTerm ||
		(args.LastLogTerm == myLastTerm && args.LastLogIndex >= myLastIdx)

	if alreadyVoted || !logOK {
		log.Printf("[%s] deny vote for %s alreadyVoted=%v logOK=%v",
			n.id, args.CandidateID, alreadyVoted, logOK)
		return reply
	}

	n.votedFor = args.CandidateID
	reply.VoteGranted = true

	n.Metrics.mu.Lock()
	n.Metrics.VotesGranted++
	n.Metrics.mu.Unlock()

	n.signalReset()
	log.Printf("[%s] grant vote to %s term=%d", n.id, args.CandidateID, n.currentTerm)
	return reply
}

func (n *Node) HandleAppendEntries(args AppendEntriesArgs) AppendEntriesReply {
	n.mu.Lock()
	defer n.mu.Unlock()

	reply := AppendEntriesReply{Term: n.currentTerm}

	if args.Term < n.currentTerm {
		return reply
	}

	if args.Term > n.currentTerm {
		n.becomeFollower(args.Term)
		reply.Term = n.currentTerm
	}

	if n.state.load() == Candidate {
		n.state.store(Follower)
		log.Printf("[%s] candidate stepping down valid leader %s appeared",
			n.id, args.LeaderID)
	}

	n.signalReset()

	n.Metrics.mu.Lock()
	n.Metrics.HeartbeatsRecvd++
	n.Metrics.CurrentLeaderID = args.LeaderID
	n.Metrics.mu.Unlock()

	reply.Success = true
	return reply
}
