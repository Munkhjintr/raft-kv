package raft

import "time"

func (rf *Raft) becomeLeader() {
	rf.state = Leader
	for _, p := range rf.peers {
		rf.nextIndex[p] = len(rf.log)
		rf.matchIndex[p] = 0
	}
	go rf.heartbeatLoop(rf.currentTerm)
}

func (rf *Raft) heartbeatLoop(term int) {
	for {
		rf.mu.Lock()
		if rf.state != Leader || rf.currentTerm != term {
			rf.mu.Unlock()
			return
		}
		rf.mu.Unlock()
		for _, p := range rf.peers {
			if p != rf.id {
				go rf.sendHeartbeat(p, term)
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (rf *Raft) sendHeartbeat(p, term int) {
	rf.mu.Lock()
	prevIndex := rf.nextIndex[p] - 1
	args := &AppendEntriesArgs{
		Term:         term,
		LeaderID:     rf.id,
		PrevLogIndex: prevIndex,
		PrevLogTerm:  rf.log[prevIndex].Term,
		LeaderCommit: rf.commitIndex,
	}
	rf.mu.Unlock()

	var reply AppendEntriesReply
	if !rf.transport.AppendEntries(p, args, &reply) {
		return
	}
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if reply.Term > rf.currentTerm {
		rf.becomeFollower(reply.Term)
	}
}