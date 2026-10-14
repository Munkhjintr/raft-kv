package raft

import (
	"net"
	"testing"
)

func TestRPCClusterReplicates(t *testing.T) {
	const n = 3
	lns := make([]net.Listener, n)
	addrs := make(map[int]string)
	peers := make([]int, n)
	for i := 0; i < n; i++ {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		lns[i] = ln
		addrs[i] = ln.Addr().String()
		peers[i] = i
	}

	nodes := make([]*Raft, n)
	for i := 0; i < n; i++ {
		nodes[i] = New(i, peers, NewRPCTransport(addrs))
		if err := ServeRPC(nodes[i], lns[i]); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for i := range nodes {
			nodes[i].Kill()
			lns[i].Close()
		}
	})

	as := collectAll(nodes)
	for _, rf := range nodes {
		rf.Start()
	}

	leader := waitForLeader(t, nodes, nil)
	nodes[leader].Propose("a")
	nodes[leader].Propose("b")
	waitApplied(t, as, nil, []interface{}{"a", "b"})
}