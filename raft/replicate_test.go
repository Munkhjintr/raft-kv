package raft

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

type applied struct {
	mu   sync.Mutex
	cmds []interface{}
}

func collect(rf *Raft) *applied {
	a := &applied{}
	go func() {
		for m := range rf.ApplyCh() {
			a.mu.Lock()
			a.cmds = append(a.cmds, m.Command)
			a.mu.Unlock()
		}
	}()
	return a
}

func (a *applied) get() []interface{} {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]interface{}, len(a.cmds))
	copy(out, a.cmds)
	return out
}

func collectAll(nodes []*Raft) []*applied {
	as := make([]*applied, len(nodes))
	for i, rf := range nodes {
		as[i] = collect(rf)
	}
	return as
}

func waitApplied(t *testing.T, as []*applied, skip map[int]bool, want []interface{}) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		ok := true
		for i, a := range as {
			if skip[i] {
				continue
			}
			if !reflect.DeepEqual(a.get(), want) {
				ok = false
			}
		}
		if ok {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	for i, a := range as {
		if !skip[i] {
			t.Logf("node %d applied %v", i, a.get())
		}
	}
	t.Fatalf("nodes did not apply %v", want)
}

func TestReplicateBasic(t *testing.T) {
	_, nodes := makeCluster(3)
	as := collectAll(nodes)
	leader := waitForLeader(t, nodes, nil)
	for _, c := range []string{"a", "b", "c"} {
		if _, _, ok := nodes[leader].Propose(c); !ok {
			t.Fatal("leader rejected proposal")
		}
	}
	waitApplied(t, as, nil, []interface{}{"a", "b", "c"})
}

func TestProposeOnFollowerRejected(t *testing.T) {
	_, nodes := makeCluster(3)
	leader := waitForLeader(t, nodes, nil)
	follower := (leader + 1) % 3
	if _, _, ok := nodes[follower].Propose("x"); ok {
		t.Fatal("follower accepted a proposal")
	}
}

func TestFollowerCatchesUp(t *testing.T) {
	net, nodes := makeCluster(3)
	as := collectAll(nodes)
	leader := waitForLeader(t, nodes, nil)
	lagging := (leader + 1) % 3

	net.SetConnected(lagging, false)
	nodes[leader].Propose("a")
	nodes[leader].Propose("b")
	waitApplied(t, as, map[int]bool{lagging: true}, []interface{}{"a", "b"})

	net.SetConnected(lagging, true)
	waitApplied(t, as, nil, []interface{}{"a", "b"})
}

func TestCommittedSurvivesLeaderFailure(t *testing.T) {
	net, nodes := makeCluster(3)
	as := collectAll(nodes)
	old := waitForLeader(t, nodes, nil)
	nodes[old].Propose("x")
	waitApplied(t, as, nil, []interface{}{"x"})

	net.SetConnected(old, false)
	skip := map[int]bool{old: true}
	newLeader := waitForLeader(t, nodes, skip)
	if _, _, ok := nodes[newLeader].Propose("y"); !ok {
		t.Fatal("new leader rejected proposal")
	}
	waitApplied(t, as, skip, []interface{}{"x", "y"})

	net.SetConnected(old, true)
	waitApplied(t, as, nil, []interface{}{"x", "y"})
}