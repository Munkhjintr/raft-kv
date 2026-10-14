package raft

import "sync"

type Network struct {
	mu        sync.Mutex
	nodes     map[int]*Raft
	connected map[int]bool
}

func NewNetwork() *Network {
	return &Network{
		nodes:     make(map[int]*Raft),
		connected: make(map[int]bool),
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

func (n *Network) Endpoint(from int) Transport {
	return &endpoint{net: n, from: from}
}

func (n *Network) reachable(from, to int) (*Raft, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.connected[from] || !n.connected[to] {
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