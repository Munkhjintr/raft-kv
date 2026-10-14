package main

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/Munkhjintr/raft-kv/kv"
)

type result struct {
	nodes      int
	clients    int
	ops        int
	throughput float64
	mean       time.Duration
	p99        time.Duration
}

func run(nodes, clients int, dur time.Duration) result {
	servers, stop := startCluster(nodes)
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
		r.mean = time.Duration(float64(clients) / r.throughput * float64(time.Second))
		r.p99 = lats[len(lats)*99/100]
	}
	return r
}