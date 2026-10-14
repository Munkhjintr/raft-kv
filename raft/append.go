package raft

import "time"

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.currentTerm
	reply.Success = false

	if args.Term < rf.currentTerm {
		return
	}
	if args.Term > rf.currentTerm {
		rf.becomeFollower(args.Term)
	}
	rf.state = Follower
	rf.lastHeartbeat = time.Now()
	reply.Term = rf.currentTerm

	if args.PrevLogIndex >= len(rf.log) || rf.log[args.PrevLogIndex].Term != args.PrevLogTerm {
		return
	}

	changed := false
	for i, e := range args.Entries {
		idx := args.PrevLogIndex + 1 + i
		if idx < len(rf.log) {
			if rf.log[idx].Term == e.Term {
				continue
			}
			rf.log = rf.log[:idx]
		}
		rf.log = append(rf.log, e)
		changed = true
	}
	if changed {
		rf.persist()
	}

	if args.LeaderCommit > rf.commitIndex {
		last := args.PrevLogIndex + len(args.Entries)
		newCommit := min(args.LeaderCommit, last)
		if newCommit > rf.commitIndex {
			rf.commitIndex = newCommit
			rf.signalApply()
		}
	}
	reply.Success = true
}