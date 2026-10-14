package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	out := flag.String("out", "bench.csv", "csv output")
	dur := flag.Duration("dur", 5*time.Second, "duration per run")
	flag.Parse()

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	fmt.Fprintln(f, "nodes,clients,ops,ops_per_sec,mean_ms,p99_ms")

	for _, nodes := range []int{1, 3, 5} {
		for _, clients := range []int{1, 4, 16, 64} {
			r := run(nodes, clients, *dur)
			fmt.Printf("nodes=%d clients=%d ops=%d %.0f ops/s mean=%v p99=%v\n",
				r.nodes, r.clients, r.ops, r.throughput, r.mean, r.p99)
			fmt.Fprintf(f, "%d,%d,%d,%.1f,%.4f,%.3f\n", r.nodes, r.clients, r.ops, r.throughput,
				float64(r.mean)/float64(time.Millisecond), float64(r.p99)/float64(time.Millisecond))
		}
	}
}