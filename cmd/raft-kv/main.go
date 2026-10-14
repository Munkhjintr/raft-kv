package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Munkhjintr/raft-kv/kv"
	"github.com/Munkhjintr/raft-kv/raft"
)

func parsePeers(s string) (map[int]string, []int, error) {
	addrs := make(map[int]string)
	var ids []int
	for _, part := range strings.Split(s, ",") {
		kvp := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kvp) != 2 {
			return nil, nil, fmt.Errorf("bad peer %q", part)
		}
		id, err := strconv.Atoi(kvp[0])
		if err != nil {
			return nil, nil, err
		}
		addrs[id] = kvp[1]
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return addrs, ids, nil
}

func main() {
	id := flag.Int("id", 0, "node id")
	peersFlag := flag.String("peers", "", "id=host:port,id=host:port,...")
	httpAddr := flag.String("http", "127.0.0.1:8000", "http listen address")
	dataDir := flag.String("data", "data", "data directory")
	flag.Parse()

	addrs, ids, err := parsePeers(*peersFlag)
	if err != nil {
		log.Fatal(err)
	}
	self, ok := addrs[*id]
	if !ok {
		log.Fatalf("id %d is not in -peers", *id)
	}
	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		log.Fatal(err)
	}

	st := raft.NewFileStorage(filepath.Join(*dataDir, fmt.Sprintf("node-%d.bin", *id)))
	rf := raft.NewWithStorage(*id, ids, raft.NewRPCTransport(addrs), st)
	srv := kv.NewServer(rf)

	ln, err := net.Listen("tcp", self)
	if err != nil {
		log.Fatal(err)
	}
	if err := raft.ServeRPC(rf, ln); err != nil {
		log.Fatal(err)
	}
	rf.Start()

	log.Printf("node %d: raft on %s, http on %s", *id, self, *httpAddr)
	log.Fatal(http.ListenAndServe(*httpAddr, srv.Handler()))
}