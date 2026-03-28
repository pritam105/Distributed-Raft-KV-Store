package raft

import (
	"testing"
	"time"
)

func waitForLeader(t *testing.T, nodes []*Node, timeout time.Duration) *Node {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var found *Node
		count := 0
		for _, nd := range nodes {
			if nd.State() == Leader {
				found = nd
				count++
			}
		}
		if count == 1 {
			return found
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no single leader found within %v", timeout)
	return nil
}

func stopAll(nodes []*Node) {
	for _, nd := range nodes {
		nd.Stop()
	}
}

func excluding(nodes []*Node, id string) []*Node {
	var out []*Node
	for _, nd := range nodes {
		if nd.ID() != id {
			out = append(out, nd)
		}
	}
	return out
}

// Test 1 - 3 nodes elect exactly one leader
func TestElection_3Nodes(t *testing.T) {
	c, nodes := NewCluster(3, 0)
	_ = c
	for _, nd := range nodes {
		nd.Start()
	}
	defer stopAll(nodes)

	leader := waitForLeader(t, nodes, 3*time.Second)
	t.Logf("leader=%s term=%d", leader.ID(), leader.Term())

	count := 0
	for _, nd := range nodes {
		if nd.State() == Leader {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 leader got %d", count)
	}
}

// Test 2 - 4 nodes elect exactly one leader
func TestElection_4Nodes(t *testing.T) {
	c, nodes := NewCluster(4, 0)
	_ = c
	for _, nd := range nodes {
		nd.Start()
	}
	defer stopAll(nodes)

	leader := waitForLeader(t, nodes, 3*time.Second)
	t.Logf("leader=%s term=%d", leader.ID(), leader.Term())
}

// Test 3 - leader crashes, new leader is elected
func TestElection_LeaderCrash(t *testing.T) {
	c, nodes := NewCluster(3, 0)
	for _, nd := range nodes {
		nd.Start()
	}
	defer stopAll(nodes)

	first := waitForLeader(t, nodes, 3*time.Second)
	firstTerm := first.Term()
	t.Logf("first leader=%s term=%d", first.ID(), firstTerm)

	c.Isolate(first.ID())
	first.Stop()

	remaining := excluding(nodes, first.ID())
	newLeader := waitForLeader(t, remaining, 4*time.Second)
	t.Logf("new leader=%s term=%d", newLeader.ID(), newLeader.Term())

	if newLeader.Term() <= firstTerm {
		t.Fatalf("new term %d must be greater than old term %d",
			newLeader.Term(), firstTerm)
	}
}

// Test 4 - follower crashes, leader stays stable
func TestElection_FollowerCrash(t *testing.T) {
	c, nodes := NewCluster(3, 0)
	for _, nd := range nodes {
		nd.Start()
	}
	defer stopAll(nodes)

	leader := waitForLeader(t, nodes, 3*time.Second)
	originalTerm := leader.Term()
	t.Logf("leader=%s term=%d", leader.ID(), originalTerm)

	for _, nd := range nodes {
		if nd.ID() != leader.ID() {
			c.Isolate(nd.ID())
			nd.Stop()
			t.Logf("crashed follower=%s", nd.ID())
			break
		}
	}

	time.Sleep(heartbeatInterval * 5)

	if leader.State() != Leader {
		t.Fatalf("leader should still be leader after follower crash")
	}
	if leader.Term() != originalTerm {
		t.Fatalf("term changed unexpectedly from %d to %d",
			originalTerm, leader.Term())
	}
	t.Logf("leader %s still stable at term=%d", leader.ID(), leader.Term())
}

// Test 5 - term increases after each re-election
func TestElection_TermIncreases(t *testing.T) {
	c, nodes := NewCluster(3, 0)
	for _, nd := range nodes {
		nd.Start()
	}
	defer stopAll(nodes)

	var lastTerm uint64
	alive := nodes

	for round := 0; round < 2; round++ {
		leader := waitForLeader(t, alive, 4*time.Second)
		if leader.Term() <= lastTerm {
			t.Fatalf("round %d term %d not greater than %d",
				round, leader.Term(), lastTerm)
		}
		lastTerm = leader.Term()
		t.Logf("round=%d leader=%s term=%d", round, leader.ID(), lastTerm)

		if round < 1 {
			c.Isolate(leader.ID())
			leader.Stop()
			alive = excluding(alive, leader.ID())
			time.Sleep(50 * time.Millisecond)
		}
	}
}
