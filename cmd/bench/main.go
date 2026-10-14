package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/Munkhjintr/raft-kv/kv"
	"github.com/Munkhjintr/raft-kv/raft"
)

type result struct {
	nodes      int
	clients    int
	ops        int
	throughput float64
	p50        time.Duration
	p99        time.Duration
}

func startCluster(n int) ([]*raft.Raft, []*kv.Server, func()) {
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
	return rafts, servers, stop
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

func run(nodes, clients int, dur time.Duration) result {
	_, servers, stop := startCluster(nodes)
	defer stop()
	waitLeader(servers)

	warm := kv.NewClerk(servers)
	if err := warm.Put("warm", "up"); err != nil {
		log.Fatal(err)
	}

	var mu sync.Mutex
	var lats []time.Duration
	var wg sync.WaitGroup
	start := time.Now()
	end := start.Add(dur)

	for c := 0; c < clients; c++ {
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			ck := kv.NewClerk(servers)
			var local []time.Duration
			for i := 0; time.Now().Before(end); i++ {
				key := fmt.Sprintf("k%d-%d", c, i%100)
				t0 := time.Now()
				if err := ck.Put(key, "value"); err != nil {
					continue
				}
				local = append(local, time.Since(t0))
			}
			mu.Lock()
			lats = append(lats, local...)
			mu.Unlock()
		}(c)
	}
	wg.Wait()
	elapsed := time.Since(start)

	sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })
	r := result{nodes: nodes, clients: clients, ops: len(lats)}
	if len(lats) > 0 {
		r.throughput = float64(len(lats)) / elapsed.Seconds()
		r.p50 = lats[len(lats)*50/100]
		r.p99 = lats[len(lats)*99/100]
	}
	return r
}

func main() {
	out := flag.String("out", "bench.csv", "csv output")
	dur := flag.Duration("dur", 5*time.Second, "duration per run")
	flag.Parse()

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	fmt.Fprintln(f, "nodes,clients,ops,ops_per_sec,p50_ms,p99_ms")

	for _, nodes := range []int{1, 3, 5} {
		for _, clients := range []int{1, 4, 16, 64} {
			r := run(nodes, clients, *dur)
			fmt.Printf("nodes=%d clients=%d ops=%d %.0f ops/s p50=%v p99=%v\n",
				r.nodes, r.clients, r.ops, r.throughput, r.p50, r.p99)
			fmt.Fprintf(f, "%d,%d,%d,%.1f,%.3f,%.3f\n", r.nodes, r.clients, r.ops, r.throughput,
				float64(r.p50)/float64(time.Millisecond), float64(r.p99)/float64(time.Millisecond))
		}
	}
}