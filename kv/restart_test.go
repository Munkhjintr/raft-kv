package kv

import (
	"testing"

	"github.com/Munkhjintr/raft-kv/raft"
)

func TestRestartAllNodes(t *testing.T) {
	const n = 3
	net := raft.NewNetwork()
	peers := []int{0, 1, 2}
	stores := make([]*raft.MemStorage, n)
	rafts := make([]*raft.Raft, n)
	servers := make([]*Server, n)
	for i := range stores {
		stores[i] = raft.NewMemStorage()
	}

	boot := func(i int) {
		rf := raft.NewWithStorage(i, peers, net.Endpoint(i), stores[i])
		net.Add(i, rf)
		rafts[i] = rf
		servers[i] = NewServer(rf)
		rf.Start()
	}
	for i := 0; i < n; i++ {
		boot(i)
	}

	ck := NewClerk(servers)
	if err := ck.Put("a", "1"); err != nil {
		t.Fatal(err)
	}
	if err := ck.Put("b", "2"); err != nil {
		t.Fatal(err)
	}
	if err := ck.Append("b", "x"); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < n; i++ {
		rafts[i].Kill()
	}
	for i := 0; i < n; i++ {
		boot(i)
	}

	for k, want := range map[string]string{"a": "1", "b": "2x"} {
		v, ok, err := ck.Get(k)
		if err != nil || !ok || v != want {
			t.Fatalf("key %s: got (%q, %v, %v), want %q", k, v, ok, err, want)
		}
	}
}