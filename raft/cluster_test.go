package raft

import (
	"testing"
	"time"
)

func makeCluster(n int) (*Network, []*Raft) {
	net := NewNetwork()
	peers := make([]int, n)
	for i := range peers {
		peers[i] = i
	}
	nodes := make([]*Raft, n)
	for i := 0; i < n; i++ {
		nodes[i] = New(i, peers, &endpoint{net: net, from: i})
		net.Add(i, nodes[i])
	}
	for _, rf := range nodes {
		rf.Start()
	}
	return net, nodes
}

func waitForLeader(t *testing.T, nodes []*Raft, skip map[int]bool) int {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		found, count := -1, 0
		for i, rf := range nodes {
			if skip[i] {
				continue
			}
			if _, isLeader := rf.GetState(); isLeader {
				found = i
				count++
			}
		}
		if count == 1 {
			return found
		}
	}
	t.Fatal("no single leader elected")
	return -1
}

func TestClusterElectsLeader(t *testing.T) {
	_, nodes := makeCluster(3)
	leader := waitForLeader(t, nodes, nil)
	term, _ := nodes[leader].GetState()
	if term < 1 {
		t.Fatalf("leader %d has term %d", leader, term)
	}
}

func TestClusterLeaderStaysStable(t *testing.T) {
	_, nodes := makeCluster(3)
	leader := waitForLeader(t, nodes, nil)
	term1, _ := nodes[leader].GetState()
	time.Sleep(time.Second)
	term2, isLeader := nodes[leader].GetState()
	if !isLeader || term1 != term2 {
		t.Fatalf("leadership changed: term %d -> %d, isLeader=%v", term1, term2, isLeader)
	}
}

func TestClusterReelectsAfterLeaderFailure(t *testing.T) {
	net, nodes := makeCluster(3)
	old := waitForLeader(t, nodes, nil)
	oldTerm, _ := nodes[old].GetState()

	net.SetConnected(old, false)
	newLeader := waitForLeader(t, nodes, map[int]bool{old: true})
	if newLeader == old {
		t.Fatal("same node re-elected while disconnected")
	}
	newTerm, _ := nodes[newLeader].GetState()
	if newTerm <= oldTerm {
		t.Fatalf("new term %d not greater than old term %d", newTerm, oldTerm)
	}

	net.SetConnected(old, true)
	final := waitForLeader(t, nodes, nil)
	if final != newLeader {
		t.Fatalf("expected leader %d to remain, got %d", newLeader, final)
	}
}

func TestClusterNoLeaderWithoutQuorum(t *testing.T) {
	net, nodes := makeCluster(3)
	leader := waitForLeader(t, nodes, nil)
	for i := range nodes {
		if i != leader {
			net.SetConnected(i, false)
		}
	}
	time.Sleep(time.Second)
	for i, rf := range nodes {
		if i == leader {
			continue
		}
		if _, isLeader := rf.GetState(); isLeader {
			t.Fatalf("node %d became leader without quorum", i)
		}
	}
}