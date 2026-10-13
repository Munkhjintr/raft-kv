package raft

import "testing"

func newTestNode() *Raft {
	return New(0, []int{0, 1, 2})
}

func TestRequestVoteGrantsFirstVote(t *testing.T) {
	rf := newTestNode()
	var reply RequestVoteReply
	rf.RequestVote(&RequestVoteArgs{Term: 1, CandidateID: 1}, &reply)
	if !reply.VoteGranted || reply.Term != 1 {
		t.Fatalf("expected vote granted in term 1, got %+v", reply)
	}
}

func TestRequestVoteRejectsStaleTerm(t *testing.T) {
	rf := newTestNode()
	rf.currentTerm = 5
	var reply RequestVoteReply
	rf.RequestVote(&RequestVoteArgs{Term: 3, CandidateID: 1}, &reply)
	if reply.VoteGranted || reply.Term != 5 {
		t.Fatalf("expected rejection with term 5, got %+v", reply)
	}
}

func TestRequestVoteOnlyOncePerTerm(t *testing.T) {
	rf := newTestNode()
	var r1, r2 RequestVoteReply
	rf.RequestVote(&RequestVoteArgs{Term: 1, CandidateID: 1}, &r1)
	rf.RequestVote(&RequestVoteArgs{Term: 1, CandidateID: 2}, &r2)
	if !r1.VoteGranted || r2.VoteGranted {
		t.Fatalf("expected first granted, second rejected: %+v %+v", r1, r2)
	}
}

func TestRequestVoteRejectsOutdatedLog(t *testing.T) {
	rf := newTestNode()
	rf.log = append(rf.log, LogEntry{Term: 2})
	var reply RequestVoteReply
	rf.RequestVote(&RequestVoteArgs{Term: 3, CandidateID: 1, LastLogIndex: 1, LastLogTerm: 1}, &reply)
	if reply.VoteGranted {
		t.Fatal("expected rejection: candidate log is older")
	}
}