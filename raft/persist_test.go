package raft

import (
	"path/filepath"
	"testing"
)

func TestRestoreTermVoteAndLog(t *testing.T) {
	st := NewMemStorage()
	peers := []int{0, 1, 2}
	rf := NewWithStorage(0, peers, nil, st)

	var vr RequestVoteReply
	rf.RequestVote(&RequestVoteArgs{Term: 4, CandidateID: 2}, &vr)
	var ar AppendEntriesReply
	rf.AppendEntries(&AppendEntriesArgs{
		Term:     4,
		LeaderID: 2,
		Entries:  []LogEntry{{Term: 4, Command: "a"}, {Term: 4, Command: "b"}},
	}, &ar)
	if !vr.VoteGranted || !ar.Success {
		t.Fatalf("setup failed: %+v %+v", vr, ar)
	}
	rf.Kill()

	rf2 := NewWithStorage(0, peers, nil, st)
	term, _ := rf2.GetState()
	if term != 4 {
		t.Fatalf("term = %d, want 4", term)
	}
	if rf2.votedFor != 2 {
		t.Fatalf("votedFor = %d, want 2", rf2.votedFor)
	}
	if len(rf2.log) != 3 || rf2.log[1].Command != "a" || rf2.log[2].Command != "b" {
		t.Fatalf("log not restored: %+v", rf2.log)
	}
}

func TestNoSecondVoteAfterRestart(t *testing.T) {
	st := NewMemStorage()
	peers := []int{0, 1, 2}
	rf := NewWithStorage(0, peers, nil, st)

	var r1 RequestVoteReply
	rf.RequestVote(&RequestVoteArgs{Term: 3, CandidateID: 1}, &r1)
	rf.Kill()

	rf2 := NewWithStorage(0, peers, nil, st)
	var r2 RequestVoteReply
	rf2.RequestVote(&RequestVoteArgs{Term: 3, CandidateID: 2}, &r2)
	if !r1.VoteGranted || r2.VoteGranted {
		t.Fatalf("expected first granted, second rejected: %+v %+v", r1, r2)
	}
}

func TestFileStorageRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.bin")
	fs := NewFileStorage(path)

	data, err := fs.Load()
	if err != nil || len(data) != 0 {
		t.Fatalf("expected empty load, got %q, %v", data, err)
	}
	if err := fs.Save([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := fs.Save([]byte("world")); err != nil {
		t.Fatal(err)
	}

	data, err = NewFileStorage(path).Load()
	if err != nil || string(data) != "world" {
		t.Fatalf("got %q, %v", data, err)
	}
}

type pcluster struct {
	net    *Network
	peers  []int
	stores []*MemStorage
	nodes  []*Raft
}

func makePersistentCluster(n int) *pcluster {
	c := &pcluster{
		net:    NewNetwork(),
		peers:  make([]int, n),
		stores: make([]*MemStorage, n),
		nodes:  make([]*Raft, n),
	}
	for i := 0; i < n; i++ {
		c.peers[i] = i
		c.stores[i] = NewMemStorage()
	}
	for i := 0; i < n; i++ {
		c.boot(i)
	}
	return c
}

func (c *pcluster) boot(i int) {
	rf := NewWithStorage(i, c.peers, c.net.Endpoint(i), c.stores[i])
	c.net.Add(i, rf)
	c.nodes[i] = rf
	rf.Start()
}

func (c *pcluster) restart(i int) {
	c.nodes[i].Kill()
	c.boot(i)
}

func TestAllNodesRestart(t *testing.T) {
	c := makePersistentCluster(3)
	as := make([]*applied, 3)
	for i := range as {
		as[i] = collect(c.nodes[i])
	}

	leader := waitForLeader(t, c.nodes, nil)
	c.nodes[leader].Propose("a")
	c.nodes[leader].Propose("b")
	waitApplied(t, as, nil, []interface{}{"a", "b"})

	for i := range c.nodes {
		c.restart(i)
		as[i] = collect(c.nodes[i])
	}

	leader = waitForLeader(t, c.nodes, nil)
	if _, _, ok := c.nodes[leader].Propose("c"); !ok {
		t.Fatal("leader rejected proposal after restart")
	}
	waitApplied(t, as, nil, []interface{}{"a", "b", "c"})
}

func TestRestartedFollowerReplaysLog(t *testing.T) {
	c := makePersistentCluster(3)
	as := make([]*applied, 3)
	for i := range as {
		as[i] = collect(c.nodes[i])
	}

	leader := waitForLeader(t, c.nodes, nil)
	c.nodes[leader].Propose("a")
	waitApplied(t, as, nil, []interface{}{"a"})

	follower := (leader + 1) % 3
	c.restart(follower)
	as[follower] = collect(c.nodes[follower])

	c.nodes[leader].Propose("b")
	waitApplied(t, as, nil, []interface{}{"a", "b"})
}