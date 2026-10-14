package main

import (
	"log"
	"net"
	"time"

	"github.com/Munkhjintr/raft-kv/kv"
	"github.com/Munkhjintr/raft-kv/raft"
)

func startCluster(n int) ([]*kv.Server, func()) {
	lns := make([]net.Listener, n)
	addrs := make(map[int]string)
	peers := make([]int, n)
	for i := 0; i < n; i++ {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatal(err)
		}
		lns[i] = ln
		addrs[i] = ln.Addr().String()
		peers[i] = i
	}
	rafts := make([]*raft.Raft, n)
	servers := make([]*kv.Server, n)
	for i := 0; i < n; i++ {
		rafts[i] = raft.New(i, peers, raft.NewRPCTransport(addrs))
		servers[i] = kv.NewServer(rafts[i])
		if err := raft.ServeRPC(rafts[i], lns[i]); err != nil {
			log.Fatal(err)
		}
	}
	for _, rf := range rafts {
		rf.Start()
	}
	stop := func() {
		for i := range rafts {
			rafts[i].Kill()
			lns[i].Close()
		}
	}
	return servers, stop
}

func waitLeader(servers []*kv.Server) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, s := range servers {
			if s.IsLeader() {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	log.Fatal("no leader")
}