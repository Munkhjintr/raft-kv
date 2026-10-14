package raft

import (
	"reflect"
	"testing"
	"time"
)

func waitConverged(t *testing.T, as []*applied, last interface{}) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		first := as[0].get()
		ok := len(first) > 0 && first[len(first)-1] == last
		for _, a := range as[1:] {
			if !reflect.DeepEqual(a.get(), first) {
				ok = false
			}
		}
		if ok {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	for i, a := range as {
		t.Logf("node %d applied %v", i, a.get())
	}
	t.Fatal("nodes did not converge")
}

func TestPartitionLeaderInMinority(t *testing.T) {
	net, nodes := makeCluster(5)
	as := collectAll(nodes)
	old := waitForLeader(t, nodes, nil)
	buddy := (old + 1) % 5

	minority := map[int]bool{old: true, buddy: true}
	var majority []int
	for i := range nodes {
		if !minority[i] {
			majority = append(majority, i)
		}
	}
	net.Partition([]int{old, buddy}, majority)

	nodes[old].Propose("lost")
	newLeader := waitForLeader(t, nodes, minority)
	if _, _, ok := nodes[newLeader].Propose("x"); !ok {
		t.Fatal("majority leader rejected proposal")
	}
	waitApplied(t, as, minority, []interface{}{"x"})

	if len(as[old].get()) != 0 {
		t.Fatalf("minority leader applied %v without quorum", as[old].get())
	}

	net.Heal()
	waitApplied(t, as, nil, []interface{}{"x"})
}

func TestNoCommitWithoutQuorum(t *testing.T) {
	net, nodes := makeCluster(5)
	as := collectAll(nodes)
	old := waitForLeader(t, nodes, nil)

	net.Partition([]int{0, 1}, []int{2, 3}, []int{4})
	nodes[old].Propose("lost")
	time.Sleep(time.Second)

	for i, a := range as {
		if len(a.get()) != 0 {
			t.Fatalf("node %d applied %v without quorum", i, a.get())
		}
	}

	net.Heal()
	leader := waitForLeader(t, nodes, nil)
	if _, _, ok := nodes[leader].Propose("ok"); !ok {
		t.Fatal("leader rejected proposal after heal")
	}
	waitConverged(t, as, "ok")
}

func TestPartitionHealConverges(t *testing.T) {
	net, nodes := makeCluster(5)
	as := collectAll(nodes)
	leader := waitForLeader(t, nodes, nil)
	nodes[leader].Propose("a")
	waitApplied(t, as, nil, []interface{}{"a"})

	net.Partition([]int{0, 1}, []int{2, 3, 4})
	time.Sleep(600 * time.Millisecond)
	net.Heal()

	leader = waitForLeader(t, nodes, nil)
	if _, _, ok := nodes[leader].Propose("b"); !ok {
		t.Fatal("leader rejected proposal after heal")
	}
	waitConverged(t, as, "b")
}