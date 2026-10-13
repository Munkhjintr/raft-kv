package raft

import "time"

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}
	if args.Term > rf.currentTerm {
		rf.becomeFollower(args.Term)
	}
	rf.state = Follower
	rf.lastHeartbeat = time.Now()
	reply.Term = rf.currentTerm
	reply.Success = true
}