package raft

import "time"

func (rf *Raft) lastLogInfo() (index, term int) {
	index = len(rf.log) - 1
	return index, rf.log[index].Term
}

func (rf *Raft) becomeFollower(term int) {
	rf.state = Follower
	rf.currentTerm = term
	rf.votedFor = -1
}

func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}

	if args.Term > rf.currentTerm {
		rf.becomeFollower(args.Term)
	}

	myIndex, myTerm := rf.lastLogInfo()
	upToDate := args.LastLogTerm > myTerm ||
		(args.LastLogTerm == myTerm && args.LastLogIndex >= myIndex)

	reply.Term = rf.currentTerm
	if (rf.votedFor == -1 || rf.votedFor == args.CandidateID) && upToDate {
		rf.votedFor = args.CandidateID
		rf.lastHeartbeat = time.Now()
		reply.VoteGranted = true
	} else {
		reply.VoteGranted = false
	}
}
