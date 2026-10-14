package kv

import (
	"sync"
	"time"

	"github.com/Munkhjintr/raft-kv/raft"
)

type waiter struct {
	ch       chan Result
	clientID int64
	seq      int64
}

type Server struct {
	mu      sync.Mutex
	rf      *raft.Raft
	store   *Store
	waiters map[int]*waiter
	lastSeq map[int64]int64
}

func NewServer(rf *raft.Raft) *Server {
	s := &Server{
		rf:      rf,
		store:   New(),
		waiters: make(map[int]*waiter),
		lastSeq: make(map[int64]int64),
	}
	go s.applyLoop()
	return s
}

func (s *Server) IsLeader() bool {
	_, ok := s.rf.GetState()
	return ok
}

func (s *Server) Submit(op Op) Result {
	s.mu.Lock()
	index, _, ok := s.rf.Propose(op)
	if !ok {
		s.mu.Unlock()
		return Result{Err: ErrWrongLeader}
	}
	w := &waiter{ch: make(chan Result, 1), clientID: op.ClientID, seq: op.Seq}
	s.waiters[index] = w
	s.mu.Unlock()

	select {
	case r := <-w.ch:
		return r
	case <-time.After(time.Second):
		s.mu.Lock()
		if s.waiters[index] == w {
			delete(s.waiters, index)
		}
		s.mu.Unlock()
		return Result{Err: ErrTimeout}
	}
}

func (s *Server) applyLoop() {
	for msg := range s.rf.ApplyCh() {
		op, ok := msg.Command.(Op)
		if !ok {
			continue
		}
		s.mu.Lock()
		res := s.execute(op)
		if w, found := s.waiters[msg.Index]; found {
			delete(s.waiters, msg.Index)
			if w.clientID != op.ClientID || w.seq != op.Seq {
				res = Result{Err: ErrLostLeader}
			}
			w.ch <- res
		}
		s.mu.Unlock()
	}
}

func (s *Server) execute(op Op) Result {
	if op.Type == OpGet {
		v, ok := s.store.Get(op.Key)
		return Result{Value: v, Ok: ok}
	}
	if op.Type != OpPut && op.Type != OpAppend && op.Type != OpDelete {
		return Result{Err: ErrUnknownOp}
	}
	if op.Seq > s.lastSeq[op.ClientID] {
		s.lastSeq[op.ClientID] = op.Seq
		switch op.Type {
		case OpPut:
			s.store.Put(op.Key, op.Value)
		case OpAppend:
			s.store.Append(op.Key, op.Value)
		case OpDelete:
			s.store.Delete(op.Key)
		}
	}
	return Result{Ok: true}
}