package raft

import "sync"

type Network struct {
	mu        sync.Mutex
	nodes     map[int]*Raft
	connected map[int]bool
	group     map[int]int
}

func NewNetwork() *Network {
	return &Network{
		nodes:     make(map[int]*Raft),
		connected: make(map[int]bool),
		group:     make(map[int]int),
	}
}

func (n *Network) Add(id int, rf *Raft) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.nodes[id] = rf
	n.connected[id] = true
}

func (n *Network) SetConnected(id int, ok bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.connected[id] = ok
}

func (n *Network) Partition(groups ...[]int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.group = make(map[int]int)
	for gi, g := range groups {
		for _, id := range g {
			n.group[id] = gi + 1
		}
	}
}

func (n *Network) Heal() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.group = make(map[int]int)
}

func (n *Network) Endpoint(from int) Transport {
	return &endpoint{net: n, from: from}
}

func (n *Network) reachable(from, to int) (*Raft, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.connected[from] || !n.connected[to] {
		return nil, false
	}
	if n.group[from] != n.group[to] {
		return nil, false
	}
	return n.nodes[to], true
}

type endpoint struct {
	net  *Network
	from int
}

func (e *endpoint) RequestVote(to int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	rf, ok := e.net.reachable(e.from, to)
	if !ok {
		return false
	}
	rf.RequestVote(args, reply)
	return true
}

func (e *endpoint) AppendEntries(to int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	rf, ok := e.net.reachable(e.from, to)
	if !ok {
		return false
	}
	rf.AppendEntries(args, reply)
	return true
}