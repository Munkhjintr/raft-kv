package kv

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Munkhjintr/raft-kv/raft"
)

func makeKV(n int) (*raft.Network, []*Server) {
	net := raft.NewNetwork()
	peers := make([]int, n)
	for i := range peers {
		peers[i] = i
	}
	rafts := make([]*raft.Raft, n)
	servers := make([]*Server, n)
	for i := 0; i < n; i++ {
		rf := raft.New(i, peers, net.Endpoint(i))
		net.Add(i, rf)
		servers[i] = NewServer(rf)
		rafts[i] = rf
	}
	for _, rf := range rafts {
		rf.Start()
	}
	return net, servers
}

func findLeader(t *testing.T, servers []*Server) int {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for i, s := range servers {
			if s.IsLeader() {
				return i
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("no leader elected")
	return -1
}

func TestPutGetDeleteThroughRaft(t *testing.T) {
	_, servers := makeKV(3)
	ck := NewClerk(servers)

	if err := ck.Put("a", "1"); err != nil {
		t.Fatal(err)
	}
	v, ok, err := ck.Get("a")
	if err != nil || !ok || v != "1" {
		t.Fatalf("got (%q, %v, %v)", v, ok, err)
	}
	if err := ck.Delete("a"); err != nil {
		t.Fatal(err)
	}
	_, ok, err = ck.Get("a")
	if err != nil || ok {
		t.Fatalf("expected key deleted, got ok=%v err=%v", ok, err)
	}
}

func TestDuplicateRequestAppliedOnce(t *testing.T) {
	_, servers := makeKV(3)
	leader := findLeader(t, servers)
	op := Op{Type: OpAppend, Key: "k", Value: "x", ClientID: 1, Seq: 1}

	for i := 0; i < 2; i++ {
		if r := servers[leader].Submit(op); r.Err != "" {
			t.Fatalf("submit %d failed: %s", i, r.Err)
		}
	}
	v, _, err := NewClerk(servers).Get("k")
	if err != nil || v != "x" {
		t.Fatalf("expected \"x\", got %q (err=%v)", v, err)
	}
}

func TestSurvivesLeaderFailure(t *testing.T) {
	net, servers := makeKV(3)
	ck := NewClerk(servers)
	if err := ck.Put("a", "1"); err != nil {
		t.Fatal(err)
	}

	old := findLeader(t, servers)
	net.SetConnected(old, false)

	if err := ck.Put("b", "2"); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{"a": "1", "b": "2"} {
		v, ok, err := ck.Get(k)
		if err != nil || !ok || v != want {
			t.Fatalf("key %s: got (%q, %v, %v), want %q", k, v, ok, err, want)
		}
	}

	net.SetConnected(old, true)
	if err := ck.Put("c", "3"); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentClients(t *testing.T) {
	_, servers := makeKV(3)
	var wg sync.WaitGroup
	for c := 0; c < 3; c++ {
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			ck := NewClerk(servers)
			for i := 0; i < 5; i++ {
				key := fmt.Sprintf("c%d-%d", c, i)
				if err := ck.Put(key, key); err != nil {
					t.Errorf("put %s: %v", key, err)
					return
				}
			}
		}(c)
	}
	wg.Wait()

	ck := NewClerk(servers)
	for c := 0; c < 3; c++ {
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("c%d-%d", c, i)
			v, ok, err := ck.Get(key)
			if err != nil || !ok || v != key {
				t.Fatalf("key %s: got (%q, %v, %v)", key, v, ok, err)
			}
		}
	}
}