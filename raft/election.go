package raft

import "time"

// lastLogInfo returns the index and term of the last log entry.
// Caller must hold rf.mu.
func (rf *Raft) lastLogInfo() (index, term int) {
	index = len(rf.log) - 1
	return index, rf.log[index].Term
}

// becomeFollower moves to a newer term. Caller must hold rf.mu.
func (rf *Raft) becomeFollower(term int) {
	rf.state = Follower
	rf.currentTerm = term
	rf.votedFor = -1
}

// RequestVote handles a vote request (Figure 2, receiver rules).
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// Rule 1: reject stale terms.
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}

	// Newer term: step down and forget the old vote.
	if args.Term > rf.currentTerm {
		rf.becomeFollower(args.Term)
	}

	// Rule 2: vote only if we haven't voted for someone else
	// AND the candidate's log is at least as up-to-date as ours.
	myIndex, myTerm := rf.lastLogInfo()
	upToDate := args.LastLogTerm > myTerm ||
		(args.LastLogTerm == myTerm && args.LastLogIndex >= myIndex)

	reply.Term = rf.currentTerm
	if (rf.votedFor == -1 || rf.votedFor == args.CandidateID) && upToDate {
		rf.votedFor = args.CandidateID
		rf.lastHeartbeat = time.Now() // granting a vote resets the election timer
		reply.VoteGranted = true
	} else {
		reply.VoteGranted = false
	}
}
