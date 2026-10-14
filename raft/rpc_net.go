package raft

import (
	"net"
	"net/rpc"
	"sync"
	"time"
)

type RPCService struct {
	rf *Raft
}

func (s *RPCService) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) error {
	s.rf.RequestVote(args, reply)
	return nil
}

func (s *RPCService) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) error {
	s.rf.AppendEntries(args, reply)
	return nil
}

func ServeRPC(rf *Raft, ln net.Listener) error {
	srv := rpc.NewServer()
	if err := srv.RegisterName("Raft", &RPCService{rf: rf}); err != nil {
		return err
	}
	go srv.Accept(ln)
	return nil
}

type RPCTransport struct {
	mu      sync.Mutex
	addrs   map[int]string
	clients map[int]*rpc.Client
}

func NewRPCTransport(addrs map[int]string) *RPCTransport {
	return &RPCTransport{addrs: addrs, clients: make(map[int]*rpc.Client)}
}

func (t *RPCTransport) client(to int) *rpc.Client {
	t.mu.Lock()
	c := t.clients[to]
	t.mu.Unlock()
	if c != nil {
		return c
	}
	conn, err := net.DialTimeout("tcp", t.addrs[to], 200*time.Millisecond)
	if err != nil {
		return nil
	}
	c = rpc.NewClient(conn)
	t.mu.Lock()
	if old := t.clients[to]; old != nil {
		t.mu.Unlock()
		c.Close()
		return old
	}
	t.clients[to] = c
	t.mu.Unlock()
	return c
}

func (t *RPCTransport) drop(to int, c *rpc.Client) {
	t.mu.Lock()
	if t.clients[to] == c {
		delete(t.clients, to)
	}
	t.mu.Unlock()
	c.Close()
}

func (t *RPCTransport) call(to int, method string, args, reply interface{}) bool {
	c := t.client(to)
	if c == nil {
		return false
	}
	done := make(chan error, 1)
	go func() { done <- c.Call(method, args, reply) }()
	select {
	case err := <-done:
		if err != nil {
			t.drop(to, c)
			return false
		}
		return true
	case <-time.After(300 * time.Millisecond):
		t.drop(to, c)
		return false
	}
}

func (t *RPCTransport) RequestVote(to int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	return t.call(to, "Raft.RequestVote", args, reply)
}

func (t *RPCTransport) AppendEntries(to int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	return t.call(to, "Raft.AppendEntries", args, reply)
}